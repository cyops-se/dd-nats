package main

import (
	"dd-nats/common/db"
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
	svcName := "dd-nats-timescale"
	_, err := ddnats.Connect(nats.DefaultURL)
	if err != nil {
		log.Printf("Exiting application due to NATS connection failure, err: %s", err.Error())
		return
	}

	if ctx := ddsvc.ProcessArgs(svcName); ctx == nil {
		return
	} else {
		if err := db.ConnectDatabase(*ctx, svcName+".db"); err != nil {
			logger.Error("Local database", "Failed to connect to local database, error: %s", err.Error())
			return
		}
	}

	db.ConfigureTypes(db.DB, &types.Log{}, &types.KeyValuePair{})

	go ddnats.SendHeartbeat(svcName)
	ddsvc.RunService(svcName, runEngine)

	log.Printf("Exiting ...")
}

func runEngine() {
	logger.Info("Microservices", "Timescale microservice running")
	emitter.Host = "192.168.0.174"
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
	}
}
