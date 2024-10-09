package main

import (
	"compress/gzip"
	"dd-nats/common/ddsvc"
	"dd-nats/common/types"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"
)

var ctx *context
var cacheMutex sync.Mutex
var gzr *gzip.Reader

type context struct {
	basedir       string
	processingdir string
	donedir       string
	faildir       string
	svc           *ddsvc.DdUsvc
}

type CacheUnpackRequest struct {
	ItemsPerSec int `json:"itemspersec"`
}

var request CacheUnpackRequest
var counter int

func main() {
	if svc := ddsvc.InitService("dd-nats-cache-unpack"); svc != nil {
		svc.RunService(runEngine)
	}

	log.Printf("Exiting ...")
}

func runEngine(svc *ddsvc.DdUsvc) {
	log.Println("Engine running ...")

	// Listen for incoming files
	ctx = initContext(".", svc)
	svc.Subscribe("usvc.cache.unpack", cacheUnpackHandler)

	request := &CacheUnpackRequest{ItemsPerSec: 10000}
	svc.Request("usvc.cache.unpack", request)
}

func processCache(fullname string, info os.FileInfo, err error) error {
	if !info.IsDir() && path.Ext(info.Name()) == ".gz" {
		ctx.svc.Info("cache unpack", "file: %s, info.Name(): %s", fullname, info.Name())
		file, _ := os.OpenFile(fullname, os.O_RDONLY, 0644)
		defer file.Close()

		gzr, _ := gzip.NewReader(file)
		defer gzr.Close()

		content, err := io.ReadAll(gzr)
		if err != nil {
			ctx.svc.Error("failed to read .gz file: %s, %s", fullname, err.Error())
			return err
		}

		var datapoints types.DataPoints
		if err = json.Unmarshal(content, &datapoints); err == nil {
			for _, dp := range datapoints {
				ctx.svc.Publish("process.actual", dp)

				counter--
				if counter <= 0 {
					time.Sleep(time.Second)
					counter = request.ItemsPerSec
				}
			}
		} else {
			ctx.svc.Error("failed to unmarshal content: %s", err.Error())
		}
	}

	return nil
}

func cacheUnpackHandler(topic string, responseTopic string, data []byte) error {
	response := types.StatusResponse{Success: true}

	// We can only handle one request at a time, which is a valid use case for TryLock()
	if !cacheMutex.TryLock() {
		err := fmt.Errorf("another request is already active, aborting this request")
		response.Success = false
		response.StatusMessage = err.Error()
		ctx.svc.Publish(responseTopic, response)
		return err
	}

	defer cacheMutex.Unlock()

	if err := json.Unmarshal(data, &request); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
		ctx.svc.Publish(responseTopic, response)
		return err
	}

	counter = request.ItemsPerSec
	if err := filepath.Walk(ctx.basedir, processCache); err != nil {
		ctx.svc.Error("Cache", "Filewalk error: %s", err.Error())
		response.Success = false
		response.StatusMessage = err.Error()
		ctx.svc.Publish(responseTopic, response)
		return err
	} else {
		ctx.svc.Publish(responseTopic, response)
	}

	return nil
}

func initContext(wdir string, svc *ddsvc.DdUsvc) *context {
	ctx := &context{basedir: path.Join(wdir, "cache"), processingdir: "processing", donedir: "done", faildir: "failed", svc: svc}
	os.MkdirAll(path.Join(ctx.basedir, ctx.processingdir), 0755)
	os.MkdirAll(path.Join(ctx.basedir, ctx.donedir), 0755)
	os.MkdirAll(path.Join(ctx.basedir, ctx.faildir), 0755)
	return ctx
}
