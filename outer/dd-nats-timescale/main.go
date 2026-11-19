package main

import (
	"dd-nats/common/cachefile"
	"dd-nats/common/ddsvc"
	"dd-nats/common/types"
	"fmt"
	"log"
	"path/filepath"
	"sync"
)

var cacheMutex sync.Mutex
var emitter TimescaleEmitter
var svc *ddsvc.DdUsvc

var topicLock sync.Mutex
var currentTopic string

func main() {
	if svc = ddsvc.InitService("dd-nats-timescale"); svc != nil {
		svc.RunService(runEngine)
	}

	log.Printf("Exiting ...")
}

func runEngine(s *ddsvc.DdUsvc) {
	svc.Info("Microservices", "Timescale microservice running")

	emitter.Host = svc.Get("host", "localhost")
	emitter.Database = svc.Get("database", "postgres")
	emitter.Port = svc.GetInt("port", 5432)
	emitter.User = svc.Get("user", "postgres")
	emitter.Password = svc.Get("password", "")
	emitter.SSLMode = svc.Get("sslmode", "disable")
	emitter.Batchsize = svc.GetInt("batchsize", 5)
	emitter.InitEmitter()
	svc.OnStop(emitter.Stop)

	registerRoutes()
	subscribeDataTopic()
	svc.Subscribe("usvc.ddnatstimescale.cache.process", processCacheHandler)
	svc.Subscribe("usvc.ddnatstimescale.event.settingschanged", settingsChangedHandler)

	// Process cached files waiting at start up
	go processCache()
}

// (Re)subscribes the configured process data topic, making sure there is only
// ever one active subscription
func subscribeDataTopic() {
	topicLock.Lock()
	defer topicLock.Unlock()

	topic := svc.Get("topic", "inner.process.actual")
	if topic == currentTopic {
		return
	}

	if currentTopic != "" {
		svc.Unsubscribe(currentTopic)
	}

	if err := svc.Subscribe(topic, emitter.ProcessDataPointHandler); err == nil {
		currentTopic = topic
	}
}

func settingsChangedHandler(subject string, responseTopic string, data []byte) error {
	subscribeDataTopic()
	return nil
}

func processCacheHandler(topic string, responseTopic string, data []byte) error {
	response := types.StatusResponse{Success: true, StatusMessage: "cache processing started"}

	// Process in the background, a big cache takes longer than any request timeout
	go processCache()

	return svc.Publish(responseTopic, response)
}

// Feeds all cache files in the cache directory to the emitter. Processed
// files are moved to cache/done (or cache/failed) so they are only processed once.
func processCache() error {
	// We can only handle one request at a time, which is a valid use case for TryLock()
	if !cacheMutex.TryLock() {
		return fmt.Errorf("another request is already active, aborting this request")
	}
	defer cacheMutex.Unlock()

	files, points, err := cachefile.ProcessDirectory(filepath.Join(svc.Context.Wdir, "cache"), emitter.ProcessMessage, func(filename string, err error) {
		svc.Error("Cache", "Failed to process cache file %s: %s", filename, err.Error())
	})

	if err != nil {
		svc.Error("Cache", "Filewalk error: %s", err.Error())
	}

	if files > 0 {
		svc.Info("Cache", "Processed %d cache file(s), %d data point(s)", files, points)
	}

	return err
}
