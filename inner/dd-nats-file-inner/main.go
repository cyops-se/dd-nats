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
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"os"
	"path"
	"time"

	"github.com/nats-io/nats.go"
)

var forwarder chan *nats.Msg = make(chan *nats.Msg, 2000)
var udpconn net.Conn
var nc *nats.Conn
var err error

var packet []byte

type context struct {
	basedir       string
	newdir        string
	processingdir string
	donedir       string
	faildir       string
}

func main() {
	svcName := "dd-nats-file-inner"
	nc, err = ddnats.Connect(nats.DefaultURL)
	if err != nil {
		log.Printf("Exiting application due to NATS connection failure, err: %s", err.Error())
		return
	}

	if ctx := ddsvc.ProcessArgs(svcName); ctx == nil {
		return
	}

	go ddnats.SendHeartbeat(svcName)
	ddsvc.RunService(svcName, runEngine)

	log.Printf("Exiting ...")
}

func runEngine() {
	log.Println("Engine running ...")

	// Watch folders for new data
	ctx := initContext(".")
	go monitorFilesystem(ctx)
}

func monitorFilesystem(ctx *context) {
	ticker := time.NewTicker(500 * time.Millisecond)

	for {
		<-ticker.C
		processDirectory(ctx, ".")
	}
}

func processDirectory(ctx *context, dirname string) {
	readdir := path.Join(ctx.basedir, ctx.newdir, dirname)
	processingdir := path.Join(ctx.basedir, ctx.processingdir, dirname)
	os.MkdirAll(processingdir, 0755)

	infos, _ := ioutil.ReadDir(readdir)
	for _, fi := range infos {
		if !fi.IsDir() {
			filename := path.Join(readdir, fi.Name())
			movename := path.Join(processingdir, fi.Name())
			if err := os.Rename(filename, movename); err == nil {
				info := &types.FileInfo{Name: fi.Name(), Path: dirname, Size: int(fi.Size()), Date: fi.ModTime()}
				data, _ := json.Marshal(info)
				nc.Publish("ui.filetransfer.request", data)
				sendFile(ctx, info) // Do it sequentially to minimize packet loss
			} else {
				// log.Printf("Failed to move file to processing area: %s, error %s", filename, err.Error())
			}
		} else {
			processDirectory(ctx, path.Join(dirname, fi.Name()))
		}
	}
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

	file, err := os.Open(filename)
	if err != nil {
		logger.Error("Filetransfer", "Failed to open %s", filename)
		return err
	}

	// Always send packets of 512KB bytes, regardless
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
			time.Sleep(100 * time.Millisecond)
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
	ctx := &context{basedir: path.Join(wdir, "outgoing"), newdir: "new", processingdir: "processing", donedir: "done", faildir: "failed"}
	os.MkdirAll(path.Join(ctx.basedir, ctx.newdir), 0755)
	os.MkdirAll(path.Join(ctx.basedir, ctx.processingdir), 0755)
	os.MkdirAll(path.Join(ctx.basedir, ctx.donedir), 0755)
	os.MkdirAll(path.Join(ctx.basedir, ctx.faildir), 0755)
	return ctx
}
