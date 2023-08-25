package main

import (
	"crypto/sha256"
	"dd-nats/common/ddnats"
	"dd-nats/common/ddsvc"
	"dd-nats/common/logger"
	"dd-nats/common/types"
	"encoding/json"
	"hash"
	"io"
	"log"
	"os"
	"path"
	"strings"

	"github.com/nats-io/nats.go"
)

var ctx *context
var err error

var packet []byte

type context struct {
	basedir       string
	processingdir string
	donedir       string
	faildir       string
}

type transferInfo struct {
	start types.FileTransferStart
	block types.FileTransferBlock
	end   types.FileTransferEnd
	file  *os.File
}

var register map[string]*transferInfo

func main() {
	if svc := ddsvc.InitService("dd-nats-file-outer"); svc != nil {
		register = make(map[string]*transferInfo)
		svc.RunService(runEngine)
	}

	log.Printf("Exiting ...")
}

func runEngine(svc *ddsvc.DdUsvc) {
	log.Println("Engine running ...")

	// Listen for incoming files
	ctx = initContext(".")
	ddnats.Subscribe("inner.file.start", fileStartHandler)
	ddnats.Subscribe("inner.file.block.*", fileBlockHandler)
	ddnats.Subscribe("inner.file.end", fileEndHandler)
}

func fileEndHandler(msg *nats.Msg) {
	var end types.FileTransferEnd
	if err := json.Unmarshal(msg.Data, &end); err != nil {
		logger.Error("File start", "Failed to unmarshal file end message: %s", err.Error())
		return
	}

	logger.Trace("File end", "End receiving file id: %s", end.TransferId)

	if entry, ok := register[end.TransferId]; ok {
		if entry.block.FileIndex == uint64(entry.start.Size) {
			fileComplete(entry)
		}
	}
}

func fileBlockHandler(msg *nats.Msg) {
	var block types.FileTransferBlock
	if err := json.Unmarshal(msg.Data, &block); err != nil {
		logger.Error("File start", "Failed to unmarshal file block message: %s", err.Error())
		return
	}

	logger.Trace("File block", "Transfer id %s, received block: %d, index: %d, size: %d", block.TransferId, block.BlockNo, block.FileIndex, block.Size)
	parts := strings.Split(msg.Subject, ".")
	id := parts[3] // block id comes last "file.block.[blockid]"
	if entry, ok := register[id]; ok {
		entry.block = block
		_, err := entry.file.Write(block.Payload)
		if err != nil {
			logger.Error("Failed to receive file", "Error writing file, err: %s", err.Error())
			fileFail(entry)
		}

		if entry.block.FileIndex == uint64(entry.start.Size) {
			fileComplete(entry)
		}
	}
}

func fileStartHandler(msg *nats.Msg) {
	var start types.FileTransferStart
	if err := json.Unmarshal(msg.Data, &start); err != nil {
		logger.Error("File start", "Failed to unmarshal file start message: %s", err.Error())
		return
	}

	entry, ok := register[start.TransferId]
	if !ok {
		entry = new(transferInfo)
		entry.start = start
		register[start.TransferId] = entry
	}

	logger.Trace("File start", "Start receiving file: %s, size: %d, id: %s", start.Name, start.Size, start.TransferId)
	filepath := path.Join(ctx.basedir, ctx.processingdir, start.Path)
	os.MkdirAll(filepath, 0755)
	entry.file, err = os.Create(path.Join(filepath, entry.start.Name))
}

func fileComplete(entry *transferInfo) {
	logger.Trace("File complete", "Closing file handle for id: %s, name: %s", entry.start.TransferId, entry.start.Name)
	entry.file.Close()
	delete(register, entry.block.TransferId)

	fromfile := path.Join(ctx.basedir, ctx.processingdir, entry.start.Path, entry.start.Name)
	topath := path.Join(ctx.basedir, ctx.donedir, entry.start.Path)
	os.MkdirAll(topath, 0755)
	tofile := path.Join(topath, entry.start.Name)
	if err := os.Rename(fromfile, tofile); err != nil {
		logger.Error("File complete", "Failed to move file: %s to done directory: %s, error: %s", fromfile, tofile, err.Error())
	}
}

func fileFail(entry *transferInfo) {
	logger.Trace("File transfer failed", "Closing file handle for id: %s, name: %s", entry.start.TransferId, entry.start.Name)
	entry.file.Close()
	delete(register, entry.block.TransferId)

	fromfile := path.Join(ctx.basedir, ctx.processingdir, entry.start.Path, entry.start.Name)
	topath := path.Join(ctx.basedir, ctx.faildir, entry.start.Path)
	os.MkdirAll(topath, 0755)
	tofile := path.Join(topath, entry.start.Name)

	if err := os.Rename(fromfile, tofile); err != nil {
		logger.Error("File complete", "Failed to move file: %s to fail directory: %s, error: %s", fromfile, tofile, err.Error())
	}
}

func calcHash(filename string) hash.Hash {
	f, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Fatal(err)
	}

	return h
}

func initContext(wdir string) *context {
	ctx := &context{basedir: path.Join(wdir, "incoming"), processingdir: "processing", donedir: "done", faildir: "failed"}
	os.MkdirAll(path.Join(ctx.basedir, ctx.processingdir), 0755)
	os.MkdirAll(path.Join(ctx.basedir, ctx.donedir), 0755)
	os.MkdirAll(path.Join(ctx.basedir, ctx.faildir), 0755)
	go createManifest(ctx)
	return ctx
}
