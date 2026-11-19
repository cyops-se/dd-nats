package app

import (
	"crypto/sha256"
	"dd-nats/common/ddsvc"
	"dd-nats/common/filetransfer"
	"dd-nats/common/types"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"time"
)

type context struct {
	BaseDir       string
	NewDir        string
	ProcessingDir string
	DoneDir       string
	FailDir       string
	BlockSize     int
	BlockDelay    time.Duration
	StableTime    time.Duration
	svc           *ddsvc.DdUsvc
}

type progress struct {
	Name    string  `json:"name"`
	Size    uint64  `json:"size"`
	Index   uint64  `json:"index"`
	Percent float64 `json:"percent"`
}

var ctx *context

// Init creates the file transfer context and directories. Must be called before
// routes are registered.
func Init(svc *ddsvc.DdUsvc) {
	ctx = initContext(svc)
}

func RunEngine(svc *ddsvc.DdUsvc) {
	svc.Info("File transfer", "Engine running ...")

	if ctx == nil {
		Init(svc)
	}

	// Watch folders for new data
	go monitorFilesystem(ctx)
	go createManifest(ctx)
	go pruneDoneDirectory(ctx)
}

func Context() *context {
	return ctx
}

func (c *context) Dir(name string) string {
	return filepath.Join(c.BaseDir, name)
}

func monitorFilesystem(ctx *context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		processDirectory(ctx, ".")
	}
}

// dirname is the path relative to the 'new' directory in forward slash format
func processDirectory(ctx *context, dirname string) {
	readdir := filepath.Join(ctx.Dir(ctx.NewDir), filepath.FromSlash(dirname))
	processingdir := filepath.Join(ctx.Dir(ctx.ProcessingDir), filepath.FromSlash(dirname))

	entries, err := os.ReadDir(readdir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			processDirectory(ctx, path.Join(dirname, entry.Name()))
			continue
		}

		fi, err := entry.Info()
		if err != nil {
			continue
		}

		// Don't pick up files that are still being written
		if time.Since(fi.ModTime()) < ctx.StableTime {
			continue
		}

		if err := os.MkdirAll(processingdir, 0755); err != nil {
			ctx.svc.Error("Filetransfer", "Failed to create directory: %s, error: %s", processingdir, err.Error())
			return
		}

		filename := filepath.Join(readdir, fi.Name())
		movename := filepath.Join(processingdir, fi.Name())
		if err := os.Rename(filename, movename); err != nil {
			// Most likely locked by another process that is still writing the file, try again later
			continue
		}

		info := &types.FileInfo{Name: fi.Name(), Path: dirname, Size: int(fi.Size()), Date: fi.ModTime()}
		ctx.svc.Event("filetransfer.request", info)
		sendFile(ctx, info) // Do it sequentially to minimize packet loss
	}
}

func sendFile(ctx *context, info *types.FileInfo) error {
	dir := info.Path
	name := info.Name
	filename := filepath.Join(ctx.Dir(ctx.ProcessingDir), filepath.FromSlash(dir), name)

	fi, err := os.Lstat(filename)
	if err != nil {
		ctx.svc.Error("Filetransfer", "Cannot find file: %s, error: %s", filename, err.Error())
		return fmt.Errorf("file not found")
	}

	if fi.IsDir() {
		ctx.svc.Error("Filetransfer", "'filename' points to a directory, not a file: %s", filename)
		return fmt.Errorf("directory, not file")
	}

	if fi.Size() == 0 {
		ctx.svc.Error("Filetransfer", "'filename' is empty: %s", filename)
		moveFile(ctx, filename, ctx.FailDir, dir, name)
		return fmt.Errorf("empty file")
	}

	file, err := os.Open(filename)
	if err != nil {
		ctx.svc.Error("Filetransfer", "Failed to open %s: %s", filename, err.Error())
		moveFile(ctx, filename, ctx.FailDir, dir, name)
		return err
	}

	size := fi.Size()
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	start := &types.FileTransferStart{Name: name, Path: dir, Size: int(size), FileTime: fi.ModTime(), TransferStart: time.Now().UTC(), TransferId: id}
	ctx.svc.Publish("file.start", start)

	content := make([]byte, ctx.BlockSize)
	block := &types.FileTransferBlock{TransferId: id, BlockNo: 0, FileIndex: 0}
	subject := fmt.Sprintf("file.block.%s", id)
	errstr := ""
	hasher := sha256.New() // calculated on the data actually sent

	p := &progress{Name: filename, Size: uint64(size), Index: 0}
	var readerr error
	for readerr == nil {
		var n int
		n, readerr = io.ReadFull(file, content)
		if readerr == io.ErrUnexpectedEOF {
			readerr = io.EOF
		}

		if n > 0 {
			hasher.Write(content[:n])
			block.Payload = content[:n]
			block.Size = uint64(n)
			block.FileIndex += block.Size
			if err := ctx.svc.Publish(subject, block); err != nil {
				errstr = ctx.svc.Error("NATS error", "Failed to publish file block, err: %s", err.Error()).Error()
				readerr = err
				break
			}

			p.Index = block.FileIndex
			p.Percent = (float64(p.Index) / float64(p.Size)) * 100.0
			ctx.svc.Event("filetransfer.progress", p)
			block.BlockNo++
			time.Sleep(ctx.BlockDelay)
		}
	}

	file.Close()

	if readerr != io.EOF && errstr == "" {
		errstr = readerr.Error()
	}

	end := types.FileTransferEnd{TransferId: id, TotalBlocks: block.BlockNo, TotalSize: block.FileIndex, Hash: hasher.Sum(nil), Error: errstr}
	ctx.svc.Publish("file.end", end)

	todir := ctx.DoneDir
	if errstr != "" {
		ctx.svc.Error("File transfer failed", "File %s ended up with an error: %s", filename, errstr)
		todir = ctx.FailDir
	}

	err = moveFile(ctx, filename, todir, dir, name)
	if err == nil {
		ctx.svc.Trace("File transfer complete", "File %s, size %d transferred", filename, size)
	}

	ctx.svc.Event("filetransfer.complete", end)
	return err
}

func moveFile(ctx *context, filename string, todir string, dir string, name string) error {
	topath := filepath.Join(ctx.Dir(todir), filepath.FromSlash(dir))
	if err := os.MkdirAll(topath, 0755); err != nil {
		return ctx.svc.Error("Failed to move file", "Failed to create directory %s, error %s", topath, err.Error())
	}

	movename := filepath.Join(topath, name)
	if err := os.Rename(filename, movename); err != nil {
		return ctx.svc.Error("Failed to move file", "Error when attempting to move file %s to %s, error %s", filename, movename, err.Error())
	}

	return nil
}

// Remove files from the done directory older than the configured retention
// time (in days, 0 = keep forever)
func pruneDoneDirectory(ctx *context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		days := ctx.svc.GetInt("done-retention-days", 0)
		if days > 0 {
			count, err := filetransfer.PruneOlderThan(ctx.Dir(ctx.DoneDir), time.Duration(days)*24*time.Hour)
			if err != nil {
				ctx.svc.Error("File transfer", "Failed to prune done directory: %s", err.Error())
			} else if count > 0 {
				ctx.svc.Info("File transfer", "%d file(s) older than %d days removed from done directory", count, days)
			}
		}

		<-ticker.C
	}
}

func initContext(svc *ddsvc.DdUsvc) *context {
	wdir := svc.Context.Wdir
	ctx := &context{
		BaseDir:       filepath.Join(wdir, "outgoing"),
		NewDir:        "new",
		ProcessingDir: "processing",
		DoneDir:       "done",
		FailDir:       "failed",
		BlockSize:     256 * 1024, // Always send blocks of 256KB bytes, regardless
		BlockDelay:    time.Duration(svc.GetInt("block-delay-ms", 200)) * time.Millisecond,
		StableTime:    time.Duration(svc.GetInt("stable-seconds", 2)) * time.Second,
		svc:           svc,
	}

	for _, dir := range []string{ctx.NewDir, ctx.ProcessingDir, ctx.DoneDir, ctx.FailDir} {
		if err := os.MkdirAll(ctx.Dir(dir), 0755); err != nil {
			svc.Error("File transfer", "Failed to create directory %s: %s", ctx.Dir(dir), err.Error())
		}
	}

	// Files left in processing from a previous run (e.g. crash) are retried
	recoverProcessing(ctx)

	return ctx
}

// Move files left in the processing directory back to new
func recoverProcessing(ctx *context) {
	root := ctx.Dir(ctx.ProcessingDir)
	filepath.Walk(root, func(fullname string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(root, fullname)
		if err != nil {
			return nil
		}

		target := filepath.Join(ctx.Dir(ctx.NewDir), rel)
		os.MkdirAll(filepath.Dir(target), 0755)
		if err := os.Rename(fullname, target); err == nil {
			ctx.svc.Info("File transfer", "Retrying file left in processing: %s", rel)
		}

		return nil
	})
}
