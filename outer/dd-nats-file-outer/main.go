package main

import (
	"bytes"
	"crypto/sha256"
	"dd-nats/common/ddsvc"
	"dd-nats/common/filetransfer"
	"dd-nats/common/types"
	"encoding/json"
	"fmt"
	"hash"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var ctx *context

type context struct {
	basedir       string
	processingdir string
	donedir       string
	faildir       string
	svc           *ddsvc.DdUsvc
}

type transferInfo struct {
	start     types.FileTransferStart
	nextBlock uint64
	received  uint64
	complete  bool // all bytes received, waiting for the end message
	file      *os.File
	filename  string // full path of the file in the processing directory
	relpath   string // path relative to the processing/done/failed directories
	hasher    hash.Hash
	lastSeen  time.Time
}

// transfers that haven't received any messages within this time are aborted
const staleTimeout = 2 * time.Minute

var register = make(map[string]*transferInfo)
var registerLock sync.Mutex

func main() {
	if svc := ddsvc.InitService("dd-nats-file-outer"); svc != nil {
		svc.RunService(runEngine)
	}

	log.Printf("Exiting ...")
}

func runEngine(svc *ddsvc.DdUsvc) {
	svc.Info("File transfer", "Engine running ...")

	// Listen for incoming files
	ctx = initContext(svc.Context.Wdir, svc)
	svc.Subscribe("inner.file.start", fileStartHandler)
	svc.Subscribe("inner.file.block.>", fileBlockHandler)
	svc.Subscribe("inner.file.end", fileEndHandler)

	svc.OnStop(func() {
		registerLock.Lock()
		defer registerLock.Unlock()
		for _, entry := range register {
			if entry.file != nil {
				entry.file.Close()
			}
		}
	})

	go purgeStaleTransfers()
	go pruneDoneDirectory()
}

func (c *context) dir(name string) string {
	return filepath.Join(c.basedir, name)
}

func fileStartHandler(topic string, responseTopic string, data []byte) error {
	var start types.FileTransferStart
	if err := json.Unmarshal(data, &start); err != nil {
		ctx.svc.Error("File start", "Failed to unmarshal file start message: %s", err.Error())
		return nil
	}

	if start.TransferId == "" || start.Name == "" {
		ctx.svc.Error("File start", "Invalid file start message, missing id or name")
		return nil
	}

	relpath, err := filetransfer.SafeJoin("", start.Path, start.Name)
	if err != nil || strings.TrimSpace(filepath.Base(relpath)) == "" {
		ctx.svc.Error("File start", "Rejected file with invalid path: '%s', name: '%s', error: %v", start.Path, start.Name, err)
		return nil
	}

	ctx.svc.Trace("File start", "Start receiving file: %s, size: %d, id: %s", start.Name, start.Size, start.TransferId)

	registerLock.Lock()
	defer registerLock.Unlock()

	if old, ok := register[start.TransferId]; ok {
		// Duplicate start message, restart the transfer
		if old.file != nil {
			old.file.Close()
		}
		delete(register, start.TransferId)
	}

	filename := filepath.Join(ctx.dir(ctx.processingdir), relpath)
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		ctx.svc.Error("File start", "Failed to create directory for file: %s, error: %s", filename, err.Error())
		return nil
	}

	file, err := os.Create(filename)
	if err != nil {
		ctx.svc.Error("File start", "Failed to create file: %s, error: %s", filename, err.Error())
		return nil
	}

	entry := &transferInfo{
		start:    start,
		file:     file,
		filename: filename,
		relpath:  relpath,
		hasher:   sha256.New(),
		lastSeen: time.Now(),
	}

	if start.Size <= 0 {
		// Nothing more to receive, just wait for the end message
		entry.complete = true
		entry.file.Close()
		entry.file = nil
	}

	register[start.TransferId] = entry
	return nil
}

func fileBlockHandler(topic string, responseTopic string, data []byte) error {
	var block types.FileTransferBlock
	if err := json.Unmarshal(data, &block); err != nil {
		ctx.svc.Error("File block", "Failed to unmarshal file block message: %s", err.Error())
		return nil
	}

	id := block.TransferId
	if id == "" {
		// block id comes last "[prefix.]file.block.[blockid]"
		parts := strings.Split(topic, ".")
		id = parts[len(parts)-1]
	}

	registerLock.Lock()
	defer registerLock.Unlock()

	entry, ok := register[id]
	if !ok || entry.complete {
		return nil
	}

	entry.lastSeen = time.Now()
	ctx.svc.Trace("File block", "Transfer id %s, received block: %d, index: %d, size: %d", id, block.BlockNo, block.FileIndex, block.Size)

	if block.BlockNo != entry.nextBlock {
		fileFail(entry, fmt.Sprintf("missing block, got %d, wanted %d", block.BlockNo, entry.nextBlock))
		return nil
	}

	size := uint64(len(block.Payload))
	if entry.received+size > uint64(entry.start.Size) || entry.received+size != block.FileIndex {
		fileFail(entry, fmt.Sprintf("unexpected block size/index, received %d + %d bytes, block index %d, file size %d", entry.received, size, block.FileIndex, entry.start.Size))
		return nil
	}

	if _, err := entry.file.Write(block.Payload); err != nil {
		fileFail(entry, fmt.Sprintf("error writing file: %s", err.Error()))
		return nil
	}

	entry.hasher.Write(block.Payload)
	entry.received += size
	entry.nextBlock++

	if entry.received == uint64(entry.start.Size) {
		// All data received, close the file and wait for the end message to verify the hash
		entry.complete = true
		if err := entry.file.Close(); err != nil {
			fileFail(entry, fmt.Sprintf("error closing file: %s", err.Error()))
		}
		entry.file = nil
	}

	return nil
}

func fileEndHandler(topic string, responseTopic string, data []byte) error {
	var end types.FileTransferEnd
	if err := json.Unmarshal(data, &end); err != nil {
		ctx.svc.Error("File end", "Failed to unmarshal file end message: %s", err.Error())
		return nil
	}

	ctx.svc.Trace("File end", "End receiving file id: %s", end.TransferId)

	registerLock.Lock()
	defer registerLock.Unlock()

	entry, ok := register[end.TransferId]
	if !ok {
		return nil
	}

	switch {
	case end.Error != "":
		fileFail(entry, fmt.Sprintf("sender reported error: %s", end.Error))
	case !entry.complete:
		fileFail(entry, fmt.Sprintf("incomplete file, received %d of %d bytes", entry.received, entry.start.Size))
	case len(end.Hash) > 0 && !bytes.Equal(end.Hash, entry.hasher.Sum(nil)):
		fileFail(entry, "SHA256 hash mismatch")
	default:
		fileComplete(entry)
	}

	return nil
}

// Must be called with registerLock held
func fileComplete(entry *transferInfo) {
	ctx.svc.Trace("File complete", "File complete, id: %s, name: %s", entry.start.TransferId, entry.start.Name)
	delete(register, entry.start.TransferId)
	moveEntry(entry, ctx.donedir)
	ctx.svc.Event("filetransfer.complete", entry.start)
}

// Must be called with registerLock held
func fileFail(entry *transferInfo, reason string) {
	ctx.svc.Error("Failed to receive file", "Transfer id: %s, name: %s, reason: %s", entry.start.TransferId, entry.start.Name, reason)
	if entry.file != nil {
		entry.file.Close()
		entry.file = nil
	}

	delete(register, entry.start.TransferId)
	moveEntry(entry, ctx.faildir)
}

func moveEntry(entry *transferInfo, todir string) {
	tofile := filepath.Join(ctx.dir(todir), entry.relpath)
	if err := os.MkdirAll(filepath.Dir(tofile), 0755); err != nil {
		ctx.svc.Error("File transfer", "Failed to create directory for: %s, error: %s", tofile, err.Error())
		return
	}

	if err := os.Rename(entry.filename, tofile); err != nil {
		ctx.svc.Error("File transfer", "Failed to move file: %s to: %s, error: %s", entry.filename, tofile, err.Error())
	}
}

// Aborts transfers where messages has been lost. A file that has been
// completely received but the end message was lost is considered done, but
// a warning is logged since the hash couldn't be verified.
func purgeStaleTransfers() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		registerLock.Lock()
		for _, entry := range register {
			if time.Since(entry.lastSeen) < staleTimeout {
				continue
			}

			if entry.complete {
				ctx.svc.Error("File transfer", "End message lost for file %s (id %s), hash not verified", entry.start.Name, entry.start.TransferId)
				fileComplete(entry)
			} else {
				fileFail(entry, fmt.Sprintf("transfer timed out, received %d of %d bytes", entry.received, entry.start.Size))
			}
		}
		registerLock.Unlock()
	}
}

// Remove files from the done directory older than the configured retention
// time (in days, 0 = keep forever)
func pruneDoneDirectory() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		if days := ctx.svc.GetInt("done-retention-days", 0); days > 0 {
			count, err := filetransfer.PruneOlderThan(ctx.dir(ctx.donedir), time.Duration(days)*24*time.Hour)
			if err != nil {
				ctx.svc.Error("File transfer", "Failed to prune done directory: %s", err.Error())
			} else if count > 0 {
				ctx.svc.Info("File transfer", "%d file(s) older than %d days removed from done directory", count, days)
			}
		}

		<-ticker.C
	}
}

func initContext(wdir string, svc *ddsvc.DdUsvc) *context {
	ctx := &context{basedir: filepath.Join(wdir, "incoming"), processingdir: "processing", donedir: "done", faildir: "failed", svc: svc}
	for _, dir := range []string{ctx.processingdir, ctx.donedir, ctx.faildir} {
		if err := os.MkdirAll(ctx.dir(dir), 0755); err != nil {
			svc.Error("File transfer", "Failed to create directory %s: %s", ctx.dir(dir), err.Error())
		}
	}

	go createManifest(ctx)
	return ctx
}
