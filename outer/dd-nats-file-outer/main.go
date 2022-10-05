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
	"net"
	"os"
	"path"
	"strings"

	"github.com/nats-io/nats.go"
)

var forwarder chan *nats.Msg = make(chan *nats.Msg, 2000)
var udpconn net.Conn
var nc *nats.Conn
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

	// svcName := "dd-nats-file-outer"
	// nc, err = ddnats.Connect(nats.DefaultURL)
	// if err != nil {
	// 	log.Printf("Exiting application due to NATS connection failure, err: %s", err.Error())
	// 	return
	// }

	// c := ddsvc.ProcessArgs(svcName)
	// if c == nil {
	// 	return
	// }

	// go ddnats.SendHeartbeat(c.Name)
	// ddsvc.RunService(c.Name, runEngine)

	// log.Printf("Exiting ...")
}

func runEngine(svc *ddsvc.DdUsvc) {
	log.Println("Engine running ...")

	// Listen for incoming files
	ctx = initContext(".")
	ddnats.Subscribe("inner.forward.file.start", fileStartHandler)
	ddnats.Subscribe("inner.forward.file.*.block", fileBlockHandler)
	ddnats.Subscribe("inner.forward.file.end", fileEndHandler)
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
			logger.Trace("File complete", "Closing file handle for id: %s", entry.start.TransferId)
			entry.file.Close()
			delete(register, end.TransferId)
		}
	}
}

func fileBlockHandler(msg *nats.Msg) {
	var block types.FileTransferBlock
	if err := json.Unmarshal(msg.Data, &block); err != nil {
		logger.Error("File start", "Failed to unmarshal file block message: %s", err.Error())
		return
	}

	logger.Trace("File block", "Received block: %d, index: %d, size: %d", block.BlockNo, block.FileIndex, block.Size)
	parts := strings.Split(msg.Subject, ".")
	id := parts[3]
	if entry, ok := register[id]; ok {
		entry.block = block
		_, err := entry.file.Write(block.Payload)
		if err != nil {
			logger.Error("Failed to receive file", "Error writing file, err: %s", err.Error())
		}

		if entry.block.FileIndex == uint64(entry.start.Size) {
			logger.Trace("File complete", "Closing file handle for id: %s", entry.start.TransferId)
			entry.file.Close()
			delete(register, block.TransferId)
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

	logger.Trace("File start", "Start receiving file: %s, size: %d", start.Name, start.Size)
	filepath := path.Join(ctx.basedir, ctx.processingdir, start.Path)
	os.MkdirAll(filepath, 0755)
	entry.file, err = os.Create(path.Join(filepath, entry.start.Name))
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
	return ctx
}
