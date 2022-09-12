package main

import (
	"dd-nats/common/ddnats"
	"dd-nats/common/ddsvc"
	"dd-nats/common/logger"
	"log"
	"os"

	"github.com/nats-io/nats.go"
)

func main() {
	svcName := "dd-nats-postgres"
	_, err := ddnats.Connect(nats.DefaultURL)
	if err != nil {
		log.Printf("Exiting application due to NATS connection failure, err: %s", err.Error())
		return
	}

	if c := ddsvc.ProcessArgs(svcName); c == nil {
		return
	} else {
		ConnectDatabase(*c)
	}

	go ddnats.SendHeartbeat(os.Args[0])
	ddsvc.RunService(svcName, runEngine)

	log.Printf("Exiting ...")
}

func runEngine() {
	logger.Info("Microservices", "Postgres microservice running")

	// Listen for incoming files
	ddnats.Subscribe("inner.system.>", processDataHandler)
	// ddnats.Subscribe("inner.forward.file.*.block", fileBlockHandler)
	// ddnats.Subscribe("inner.forward.file.end", fileEndHandler)
}

func processDataHandler(msg *nats.Msg) {
	data := string(msg.Data)
	log.Printf("Process data message received on subject: %s, %s", msg.Subject, data)
}
