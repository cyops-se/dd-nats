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
	"strconv"
	"sync"
)

var cacheMutex sync.Mutex
var gzr *gzip.Reader

type CacheUnpackRequest struct {
	ItemsPerSec int `json:"itemspersec"`
}

var request CacheUnpackRequest
var counter int

var emitter TimescaleEmitter
var svc *ddsvc.DdUsvc

func main() {
	if svc = ddsvc.InitService("dd-nats-timescale"); svc != nil {
		svc.RunService(runEngine)
	}

	log.Printf("Exiting ...")
}

func runEngine(svc *ddsvc.DdUsvc) {
	svc.Info("Microservices", "Timescale microservice running")
	registerRoutes()

	emitter.Host = svc.Get("host", "localhost")
	emitter.Database = svc.Get("database", "postgres")
	emitter.Port, _ = strconv.Atoi(svc.Get("port", "5432"))
	emitter.User = svc.Get("user", "postgres")
	emitter.Password = svc.Get("password", "")
	emitter.Batchsize, _ = strconv.Atoi(svc.Get("batchsize", "5"))
	emitter.InitEmitter()

	topic := svc.Get("topic", "inner.process.actual")
	svc.Subscribe(topic, emitter.ProcessDataPointHandler)
	svc.Subscribe("usvc.ddnatstimescale.cache.process", processCacheHandler)
	svc.Subscribe("usvc.ddnatstimescale.event.settingschanged", settingsChangedHandler)

	request := &CacheUnpackRequest{ItemsPerSec: 5000}
	svc.Request("usvc.ddnatstimescale.cache.process", request)
}

func settingsChangedHandler(subject string, responseTopic string, data []byte) error {
	topic := svc.Get("topic", "inner.process.actual")
	return svc.Subscribe(topic, emitter.ProcessDataPointHandler)
}

func processCacheHandler(topic string, responseTopic string, data []byte) error {
	response := types.StatusResponse{Success: true}

	// We can only handle one request at a time, which is a valid use case for TryLock()
	if !cacheMutex.TryLock() {
		err := fmt.Errorf("another request is already active, aborting this request")
		response.Success = false
		response.StatusMessage = err.Error()
		svc.Publish(responseTopic, response)
		return err
	}

	defer cacheMutex.Unlock()

	if err := json.Unmarshal(data, &request); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
		svc.Publish(responseTopic, response)
		return err
	}

	counter = request.ItemsPerSec
	if err := filepath.Walk("cache", processCache); err != nil {
		svc.Error("Cache", "Filewalk error: %s", err.Error())
		response.Success = false
		response.StatusMessage = err.Error()
		svc.Publish(responseTopic, response)
		return err
	} else {
		svc.Publish(responseTopic, response)
	}

	return nil
}

func processCache(fullname string, info os.FileInfo, err error) error {
	if !info.IsDir() && path.Ext(info.Name()) == ".gz" {
		svc.Info("cache unpack", "file: %s, info.Name(): %s", fullname, info.Name())
		file, _ := os.OpenFile(fullname, os.O_RDONLY, 0644)
		defer file.Close()

		gzr, _ := gzip.NewReader(file)
		defer gzr.Close()

		content, err := io.ReadAll(gzr)
		if err != nil {
			svc.Error("failed to read .gz file: %s, %s ... continuing", fullname, err.Error())
			return nil
		}

		var datapoints types.DataPoints
		if err = json.Unmarshal(content, &datapoints); err == nil {
			for _, dp := range datapoints {
				emitter.ProcessMessage(dp)

				// svc.Publish("process.filtered", dp)

				// counter--
				// if counter <= 0 {
				// 	time.Sleep(time.Second)
				// 	counter = request.ItemsPerSec
				// }
			}
		} else {
			svc.Error("failed to unmarshal content: %s", err.Error())
		}
	}

	return nil
}
