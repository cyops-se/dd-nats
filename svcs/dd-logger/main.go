package main

import (
	"dd-nats/common/ddsvc"
	"dd-nats/common/types"
	"encoding/json"
	"log"
)

type categoryRequest struct {
	Category string `json:"category"`
}

type logResponse struct {
	types.StatusResponse
	Entries []types.Log `json:"items"`
}

var entries []types.Log
var empty types.Log
var svc *ddsvc.DdUsvc

func main() {
	if svc = ddsvc.InitService("dd-logger"); svc != nil {
		svc.RunService(runEngine)
	}

	log.Printf("Exiting ...")
}

func runEngine(svc *ddsvc.DdUsvc) {
	entries = make([]types.Log, 0)

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
	if err := json.Unmarshal(data, &entry); err == nil {
		entries = append(entries, entry) // enqueue new entry
		for len(entries) > 1000 {
			entries[0] = empty
			entries = entries[1:] // dequeue all entries exceeding 1000
		}
	} else {
		log.Println("Failed to unmarshal log entry:", err.Error())
	}

	return nil
}

func getAllLogs(topic string, responseTopic string, data []byte) error {
	response := &logResponse{Entries: entries}
	response.Success = true
	return svc.Publish(responseTopic, response)
}

func getCategory(topic string, responseTopic string, data []byte) error {
	var request categoryRequest
	response := &logResponse{}
	if err := json.Unmarshal(data, &request); err != nil {
		response.StatusMessage = err.Error()
	} else {
		response.Success = true
		for _, item := range entries {
			if item.Category == request.Category {
				response.Entries = append(response.Entries, item)
			}
		}
	}

	return svc.Publish(responseTopic, response)
}
