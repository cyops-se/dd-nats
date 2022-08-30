package main

import (
	"crypto/sha256"
	"dd-nats/common/ddnats"
	"dd-nats/common/ddsvc"
	"dd-nats/common/logger"
	"dd-nats/common/types"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"path"
	"strings"
	"time"

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
	svcName := "dd-nats-file-inner"
	nc, err = ddnats.Connect(nats.DefaultURL)
	if err != nil {
		log.Printf("Exiting application due to NATS connection failure, err: %s", err.Error())
		return
	}

	if c := ddsvc.ProcessArgs(svcName); c == nil {
		return
	}

	register = make(map[string]*transferInfo)

	go ddnats.SendHeartbeat(os.Args[0], nc)
	ddsvc.RunService(svcName, runEngine)

	log.Printf("Exiting ...")
}

func runEngine() {
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

func sendFile(ctx *context, info *types.FileInfo) error {
	dir := info.Path
	name := info.Name
	filename := path.Join(ctx.basedir, ctx.processingdir, dir, name)

	fi, err := os.Lstat(filename)
	if err != nil {
		logger.Error("Filetransfer", "Cannot find file: %s, error: %s", filename, err.Error())
		return fmt.Errorf("file not found")
	}

	if fi.IsDir() {
		logger.Error("Filetransfer", "'filename' points to a directory, not a file: %s", filename)
		return fmt.Errorf("directory, not file")
	}

	if fi.Size() == 0 {
		logger.Error("Filetransfer", "'filename' is empty:", filename)
		return fmt.Errorf("empty file")
	}

	id := fmt.Sprintf("%d", rand.Int())
	start := &types.FileTransferStart{Name: name, Path: dir, Size: info.Size, TransferStart: time.Now().UTC(), TransferId: id}
	ddnats.Publish("forward.file.start", start)

	hash := calcHash(filename)
	hashvalue := hash.Sum(nil)
	// header := fmt.Sprintf("DD-FILETRANSFER BEGIN v2 %s %s %d %x", name, dir, info.Size, hash.Sum(nil)) // :filename:directory:size:hash:

	file, err := os.Open(filename)
	if err != nil {
		logger.Error("Filetransfer", "Failed to open %s", filename)
		return err
	}

	// Always send packets of 1200 bytes, regardless
	content := make([]byte, 512*1024)
	n := 0
	block := &types.FileTransferBlock{TransferId: id, BlockNo: 0, FileIndex: 0}
	subject := fmt.Sprintf("forward.file.%s.block", id)
	errstr := ""

	for err == nil {
		// each message starts with a 4 byte sequence number, then 4 bytes of size of payload, then payload
		n, err = file.Read(content)
		if n > 0 {
			block.Payload = content[:n]
			block.Size = uint64(n)
			block.FileIndex += block.Size
			if err = ddnats.Publish(subject, block); err != nil {
				errstr = logger.Error("NATS error", "Failed to publish file block, err: %s", err.Error()).Error()
			}

			block.BlockNo++
			block.FileIndex += uint64(n)
			time.Sleep(10 * time.Millisecond)

			// if block.BlockNo%1000 == 0 {
			// 	percent := float64(block.FileIndex) / float64(info.Size) * 100.0
			// 	progress := &types.FileProgress{File: info, TotalSent: int(block.FileIndex), PercentDone: percent}
			// 	ddnats.Publish("ui.filetransfer.progress", progress)
			// 	time.Sleep(10 * time.Millisecond)
			// }
		}
	}

	file.Close()
	end := types.FileTransferEnd{TransferId: id, TotalBlocks: block.BlockNo, TotalSize: block.FileIndex, Hash: hashvalue, Error: errstr}
	ddnats.Publish("forward.file.end", end)

	todir := path.Join(ctx.basedir, ctx.donedir, dir)
	if err != nil && err != io.EOF {
		logger.Trace("File transfer failed", "Ended up with an error: %s", err.Error())
		todir = path.Join(ctx.basedir, ctx.faildir, dir)
	}

	os.MkdirAll(todir, 0755)

	movename := path.Join(todir, name)
	if err = os.Rename(filename, movename); err == nil {
		logger.Trace("File transfer complete", "File %s, size %d transferred, err: %s", filename, info.Size, errstr)
	} else {
		logger.Error("Failed to move file", "Error when attempting to move file after file was transferred, file %s, size %d, error %s", filename, info.Size, err.Error())
	}

	ddnats.Publish("ui.filetransfer.complete", end)

	time.Sleep(time.Millisecond)

	return err
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
