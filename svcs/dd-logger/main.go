package main

import (
	"dd-nats/common/ddnats"
	"dd-nats/common/ddsvc"
	"dd-nats/common/types"
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"
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

func main() {
	if svc := ddsvc.InitService("dd-logger"); svc != nil {
		svc.RunService(runEngine)
	}

	log.Printf("Exiting ...")
}

func runEngine(svc *ddsvc.DdUsvc) {
	entries = make([]types.Log, 0)

	// Capture system logs
	ddnats.Subscribe("system.log.>", logMessageHandler)
	ddnats.Subscribe("inner.system.log.>", logMessageHandler)

	// Service methods
	ddnats.Subscribe("usvc.logs.getall", getAllLogs)
	ddnats.Subscribe("usvc.logs.getcategory", getCategory)

	log.Println("Logging service started!")
}

func logMessageHandler(nmsg *nats.Msg) {
	var entry types.Log
	if err := json.Unmarshal(nmsg.Data, &entry); err == nil {
		entries = append(entries, entry) // enqueue new entry
		for len(entries) > 1000 {
			entries[0] = empty
			entries = entries[1:] // dequeue all entries exceeding 1000
		}
	} else {
		log.Println("Failed to unmarshal log entry:", err.Error())
	}
}

func getAllLogs(nmsg *nats.Msg) {
	response := &logResponse{Entries: entries}
	response.Success = true
	ddnats.Respond(nmsg, response)
}

func getCategory(nmsg *nats.Msg) {
	var request categoryRequest
	response := &logResponse{}
	if err := json.Unmarshal(nmsg.Data, &request); err != nil {
		response.StatusMessage = err.Error()
	} else {
		response.Success = true
		for _, item := range entries {
			if item.Category == request.Category {
				response.Entries = append(response.Entries, item)
			}
		}
	}

	ddnats.Respond(nmsg, response)
}
