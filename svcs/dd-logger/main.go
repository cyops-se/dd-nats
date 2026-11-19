package main

import (
	"dd-nats/common/ddsvc"
	"dd-nats/common/types"
	"encoding/json"
	"log"
	"sync"
)

const maxEntries = 1000

type categoryRequest struct {
	Category string `json:"category"`
}

type logResponse struct {
	types.StatusResponse
	Entries []types.Log `json:"items"`
}

// entries is written and read from several subscriptions concurrently
var entries []types.Log
var entriesLock sync.Mutex
var svc *ddsvc.DdUsvc

func main() {
	if svc = ddsvc.InitService("dd-logger"); svc != nil {
		svc.RunService(runEngine)
	}

	log.Printf("Exiting ...")
}

func runEngine(s *ddsvc.DdUsvc) {
	entries = make([]types.Log, 0, maxEntries)

	// Capture system logs
	svc.Subscribe("system.log.info", logMessageHandler)
	svc.Subscribe("system.log.error", logMessageHandler)
	svc.Subscribe("system.log.trace", logMessageHandler)
	svc.Subscribe("inner.system.log.>", logMessageHandler)

	// Service methods
	svc.Subscribe("usvc.logs.getall", getAllLogs)
	svc.Subscribe("usvc.logs.getcategory", getCategory)

	log.Println("Logging service started!")
}

func logMessageHandler(topic string, responseTopic string, data []byte) error {
	var entry types.Log
	if err := json.Unmarshal(data, &entry); err != nil {
		log.Println("Failed to unmarshal log entry:", err.Error())
		return nil
	}

	entriesLock.Lock()
	defer entriesLock.Unlock()

	if len(entries) >= maxEntries {
		// drop the oldest entries, reusing the underlying array
		n := copy(entries, entries[len(entries)-maxEntries+1:])
		entries = entries[:n]
	}

	entries = append(entries, entry)
	return nil
}

func getAllLogs(topic string, responseTopic string, data []byte) error {
	entriesLock.Lock()
	response := &logResponse{Entries: append([]types.Log(nil), entries...)}
	entriesLock.Unlock()

	response.Success = true
	return svc.Publish(responseTopic, response)
}

func getCategory(topic string, responseTopic string, data []byte) error {
	var request categoryRequest
	response := &logResponse{Entries: []types.Log{}}
	if err := json.Unmarshal(data, &request); err != nil {
		response.StatusMessage = err.Error()
	} else {
		response.Success = true

		entriesLock.Lock()
		for _, item := range entries {
			if item.Category == request.Category {
				response.Entries = append(response.Entries, item)
			}
		}
		entriesLock.Unlock()
	}

	return svc.Publish(responseTopic, response)
}
