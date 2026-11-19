package main

import (
	"dd-nats/common/cachefile"
	"dd-nats/common/ddsvc"
	"dd-nats/common/types"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var cacheMutex sync.Mutex
var svc *ddsvc.DdUsvc
var cachedir string

type CacheUnpackRequest struct {
	ItemsPerSec int `json:"itemspersec"`
}

var topic string

func main() {
	flag.StringVar(&topic, "topic", "process.filtered", "specifies the topic to use when publishing to the message broker")

	if svc = ddsvc.InitService("dd-nats-cache-unpack"); svc != nil {
		svc.RunService(runEngine)
	}

	log.Printf("Exiting ...")
}

func runEngine(s *ddsvc.DdUsvc) {
	svc.Info("Cache unpack", "Engine running ...")

	cachedir = filepath.Join(svc.Context.Wdir, "cache")
	if err := os.MkdirAll(cachedir, 0755); err != nil {
		svc.Error("Cache unpack", "Failed to create cache directory %s: %s", cachedir, err.Error())
	}

	svc.Subscribe("usvc.cache.unpack", cacheUnpackHandler)

	// Process any files waiting in the cache directory at start up. Processed
	// files are moved to cache/done (or cache/failed) so they are only processed once.
	go unpack(CacheUnpackRequest{ItemsPerSec: svc.GetInt("itemspersec", 5000)})
}

func unpack(request CacheUnpackRequest) error {
	// We can only handle one request at a time, which is a valid use case for TryLock()
	if !cacheMutex.TryLock() {
		return fmt.Errorf("another request is already active, aborting this request")
	}
	defer cacheMutex.Unlock()

	counter := request.ItemsPerSec
	files, points, err := cachefile.ProcessDirectory(cachedir, func(dp types.DataPoint) {
		svc.Publish(topic, dp)

		if request.ItemsPerSec > 0 {
			counter--
			if counter <= 0 {
				time.Sleep(time.Second)
				counter = request.ItemsPerSec
			}
		}
	}, func(filename string, err error) {
		svc.Error("Cache unpack", "Failed to process file %s: %s", filename, err.Error())
	})

	if files > 0 {
		svc.Info("Cache unpack", "Processed %d file(s), %d data point(s)", files, points)
	}

	return err
}

func cacheUnpackHandler(subject string, responseTopic string, data []byte) error {
	response := types.StatusResponse{Success: true}

	var request CacheUnpackRequest
	if err := json.Unmarshal(data, &request); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
		return svc.Publish(responseTopic, response)
	}

	// Respond immediately and process in the background, processing a big
	// cache takes much longer than any request timeout
	go func() {
		if err := unpack(request); err != nil {
			svc.Error("Cache unpack", "Unpack request failed: %s", err.Error())
		}
	}()

	response.StatusMessage = "cache unpack started"
	return svc.Publish(responseTopic, response)
}
