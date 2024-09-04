package main

import (
	"crypto/sha256"
	"dd-nats/common/ddsvc"
	"dd-nats/common/types"
	"encoding/json"
	"hash"
	"io"
	"log"
	"os"
	"path"
	"strings"
)

var ctx *context
var err error

var packet []byte

type context struct {
	basedir       string
	processingdir string
	donedir       string
	faildir       string
	svc           *ddsvc.DdUsvc
}

type transferInfo struct {
	start  types.FileTransferStart
	block  types.FileTransferBlock
	end    types.FileTransferEnd
	file   *os.File
	failed bool
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
	ctx = initContext(".", svc)
	svc.Subscribe("inner.file.start", fileStartHandler)
	svc.Subscribe("inner.file.block.>", fileBlockHandler)
	svc.Subscribe("inner.file.end", fileEndHandler)
}

func fileEndHandler(topic string, responseTopic string, data []byte) error {
	var end types.FileTransferEnd
	if err := json.Unmarshal(data, &end); err != nil {
		ctx.svc.Error("File end", "Failed to unmarshal file end message: %s", err.Error())
		return nil
	}

	ctx.svc.Trace("File end", "End receiving file id: %s", end.TransferId)

	if entry, ok := register[end.TransferId]; ok {
		if !entry.failed && entry.block.FileIndex == uint64(entry.start.Size) {
			fileComplete(entry)
		} else {
			ctx.svc.Trace("File end", "FileIndex (%d) != size (%d): %s", entry.block.FileIndex, entry.start.Size, end.TransferId)
		}
	}

	return nil
}

func fileBlockHandler(topic string, responseTopic string, data []byte) error {
	var block types.FileTransferBlock
	if err := json.Unmarshal(data, &block); err != nil {
		ctx.svc.Error("File block", "Failed to unmarshal file block message: %s", err.Error())
		ctx.svc.Trace("File block", "%s", string(data))
		return nil
	}

	parts := strings.Split(topic, ".")
	id := parts[3] // block id comes last "file.block.[blockid]"
	if entry, ok := register[id]; ok {
		if entry.failed {
			delete(register, id)
			printEntryn()
			return nil
		}

		ctx.svc.Trace("File block", "Transfer id %s, received block: %d, index: %d, size: %d", block.TransferId, block.BlockNo, block.FileIndex, block.Size)

		if block.BlockNo > 0 && block.BlockNo != entry.block.BlockNo+1 {
			ctx.svc.Error("Failed to receive file", "Missing block, got %d, wanted %d", block.BlockNo, entry.block.BlockNo+1)
			fileFail(entry)
			return nil
		}

		entry.block = block
		_, err := entry.file.Write(block.Payload)
		if err != nil {
			ctx.svc.Error("Failed to receive file", "Error writing file, err: %s", err.Error())
			fileFail(entry)
		}

		if entry.block.FileIndex == uint64(entry.start.Size) {
			fileComplete(entry)
		}
	}

	return nil
}

func fileStartHandler(topic string, responseTopic string, data []byte) error {
	var start types.FileTransferStart
	if err := json.Unmarshal(data, &start); err != nil {
		ctx.svc.Error("File start", "Failed to unmarshal file start message: %s", err.Error())
		return nil
	}

	entry, ok := register[start.TransferId]
	if !ok {
		entry = new(transferInfo)
		entry.start = start
		register[start.TransferId] = entry
	}

	ctx.svc.Trace("File start", "Start receiving file: %s, size: %d, id: %s", start.Name, start.Size, start.TransferId)
	filepath := path.Join(ctx.basedir, ctx.processingdir, start.Path)
	os.MkdirAll(filepath, 0755)
	entry.file, err = os.Create(path.Join(filepath, entry.start.Name))
	return nil
}

func fileComplete(entry *transferInfo) {
	ctx.svc.Trace("File complete", "Closing file handle for id: %s, name: %s", entry.start.TransferId, entry.start.Name)
	entry.file.Close()
	delete(register, entry.block.TransferId)

	fromfile := path.Join(ctx.basedir, ctx.processingdir, entry.start.Path, entry.start.Name)
	topath := path.Join(ctx.basedir, ctx.donedir, entry.start.Path)
	os.MkdirAll(topath, 0755)
	tofile := path.Join(topath, entry.start.Name)
	if err := os.Rename(fromfile, tofile); err != nil {
		ctx.svc.Error("File complete", "Failed to move file: %s to done directory: %s, error: %s", fromfile, tofile, err.Error())
	}
}

func fileFail(entry *transferInfo) {
	ctx.svc.Trace("File transfer failed", "Closing file handle for id: %s, name: %s", entry.start.TransferId, entry.start.Name)
	entry.failed = true
	entry.file.Close()
	delete(register, entry.block.TransferId)

	fromfile := path.Join(ctx.basedir, ctx.processingdir, entry.start.Path, entry.start.Name)
	topath := path.Join(ctx.basedir, ctx.faildir, entry.start.Path)
	os.MkdirAll(topath, 0755)
	tofile := path.Join(topath, entry.start.Name)

	if err := os.Rename(fromfile, tofile); err != nil {
		ctx.svc.Error("File complete", "Failed to move file: %s to fail directory: %s, error: %s", fromfile, tofile, err.Error())
	}
}

func printEntryn() {
	ctx.svc.Trace("File register", "Printing entry")
	for k, v := range register {
		ctx.svc.Trace("File register", "Key: %s, Name: %s", k, v.file.Name())
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

func initContext(wdir string, svc *ddsvc.DdUsvc) *context {
	ctx := &context{basedir: path.Join(wdir, "incoming"), processingdir: "processing", donedir: "done", faildir: "failed", svc: svc}
	os.MkdirAll(path.Join(ctx.basedir, ctx.processingdir), 0755)
	os.MkdirAll(path.Join(ctx.basedir, ctx.donedir), 0755)
	os.MkdirAll(path.Join(ctx.basedir, ctx.faildir), 0755)
	go createManifest(ctx)
	return ctx
}
