package app

import (
	"crypto/sha256"
	"dd-nats/common/ddnats"
	"dd-nats/common/ddsvc"
	"dd-nats/common/logger"
	"dd-nats/common/types"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"time"
)

type context struct {
	BaseDir       string
	NewDir        string
	ProcessingDir string
	DoneDir       string
	FailDir       string
}

type progress struct {
	Name    string  `json:"name"`
	Size    uint64  `json:"size"`
	Index   uint64  `json:"index"`
	Percent float64 `json:"percent"`
}

var ctx *context

func RunEngine(svc *ddsvc.DdUsvc) {
	log.Println("Engine running ...")

	// Watch folders for new data
	ctx = initContext(svc.Context.Wdir)
	go monitorFilesystem(ctx)
	go createManifest(ctx)
}

func Context() *context {
	return ctx
}

func monitorFilesystem(ctx *context) {
	ticker := time.NewTicker(500 * time.Millisecond)

	for {
		<-ticker.C
		processDirectory(ctx, ".")
	}
}

func processDirectory(ctx *context, dirname string) {
	readdir := path.Join(ctx.BaseDir, ctx.NewDir, dirname)
	processingdir := path.Join(ctx.BaseDir, ctx.ProcessingDir, dirname)
	os.MkdirAll(processingdir, 0755)

	infos, _ := ioutil.ReadDir(readdir)
	for _, fi := range infos {
		if !fi.IsDir() {
			filename := path.Join(readdir, fi.Name())
			movename := path.Join(processingdir, fi.Name())
			if err := os.Rename(filename, movename); err == nil {
				info := &types.FileInfo{Name: fi.Name(), Path: dirname, Size: int(fi.Size()), Date: fi.ModTime()}
				ddnats.Event("filetransfer.request", info)
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
	filename := path.Join(ctx.BaseDir, ctx.ProcessingDir, dir, name)

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
		logger.Error("Filetransfer", "'filename' is empty: %s", filename)
		return fmt.Errorf("empty file")
	}

	id := fmt.Sprintf("%d", time.Now().UnixNano())
	start := &types.FileTransferStart{Name: name, Path: dir, Size: info.Size, TransferStart: time.Now().UTC(), TransferId: id}
	ddnats.Publish("file.start", start)

	hash := calcHash(filename)
	hashvalue := hash.Sum(nil)

	file, err := os.Open(filename)
	if err != nil {
		logger.Error("Filetransfer", "Failed to open %s", filename)
		return err
	}

	// Always send blocks of 256KB bytes, regardless
	content := make([]byte, 256*1024)
	n := 0
	block := &types.FileTransferBlock{TransferId: id, BlockNo: 0, FileIndex: 0}
	subject := fmt.Sprintf("file.block.%s", id)
	errstr := ""

	p := &progress{Name: filename, Size: uint64(info.Size), Index: block.FileIndex}
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

			p.Index = block.FileIndex
			p.Percent = (float64(p.Index) / float64(p.Size)) * 100.0
			ddnats.Event("filetransfer.progress", p)
			block.BlockNo++
			time.Sleep(200 * time.Millisecond)
		}
	}

	file.Close()
	end := types.FileTransferEnd{TransferId: id, TotalBlocks: block.BlockNo, TotalSize: block.FileIndex, Hash: hashvalue, Error: errstr}
	ddnats.Publish("file.end", end)

	todir := path.Join(ctx.BaseDir, ctx.DoneDir, dir)
	if err != nil && err != io.EOF {
		logger.Trace("File transfer failed", "Ended up with an error: %s", err.Error())
		todir = path.Join(ctx.BaseDir, ctx.FailDir, dir)
	}

	os.MkdirAll(todir, 0755)

	movename := path.Join(todir, name)
	if err = os.Rename(filename, movename); err == nil {
		logger.Trace("File transfer complete", "File %s, size %d transferred", filename, info.Size)
	} else {
		logger.Error("Failed to move file", "Error when attempting to move file after file was transferred, file %s, size %d, error %s", filename, info.Size, err.Error())
	}

	ddnats.Event("filetransfer.complete", end)

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
	ctx := &context{BaseDir: path.Join(wdir, "outgoing"), NewDir: "new", ProcessingDir: "processing", DoneDir: "done", FailDir: "failed"}
	os.MkdirAll(path.Join(ctx.BaseDir, ctx.NewDir), 0755)
	os.MkdirAll(path.Join(ctx.BaseDir, ctx.ProcessingDir), 0755)
	os.MkdirAll(path.Join(ctx.BaseDir, ctx.DoneDir), 0755)
	os.MkdirAll(path.Join(ctx.BaseDir, ctx.FailDir), 0755)
	return ctx
}
