package main

import (
	"dd-nats/common/ddnats"
	"dd-nats/common/ddsvc"
	"dd-nats/common/logger"
	"dd-nats/common/types"
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"
)

var emitter TimescaleEmitter

func main() {
	if svc := ddsvc.InitService("dd-nats-timescale"); svc != nil {
		svc.RunService(runEngine)
	}

	log.Printf("Exiting ...")
}

func runEngine(svc *ddsvc.DdUsvc) {
	logger.Info("Microservices", "Timescale microservice running")
	emitter.Host = "localhost"
	emitter.Database = "postgres"
	emitter.Port = 5432
	emitter.User = "postgres"
	emitter.Password = "hemligt"
	emitter.Batchsize = 5
	emitter.InitEmitter()

	// Listen for incoming process data
	ddnats.Subscribe("inner.forward.process", processDataHandler)
}

func processDataHandler(nmsg *nats.Msg) {
	var msg types.DataPointSample
	if err := json.Unmarshal(nmsg.Data, &msg); err == nil {
		for _, dp := range msg.Points {
			emitter.ProcessMessage(dp)
		}
	} else {
		logger.Error("Timescale server", "Failed to unmarshal process data: %s", err.Error())
	}
}
