package main

import (
	"dd-nats/common/ddnats"
	"dd-nats/common/ddsvc"
	"dd-nats/common/logger"
	"dd-nats/inner/dd-nats-opcda/data"
	"dd-nats/inner/dd-nats-opcda/routes"
	"log"
	"net"

	"github.com/nats-io/nats.go"
)

var forwarder chan *nats.Msg = make(chan *nats.Msg, 2000)
var udpconn net.Conn

var packet []byte

func main() {
	svcName := "dd-nats-opcda"
	_, err := ddnats.Connect(nats.DefaultURL)
	if err != nil {
		log.Printf("Exiting application due to NATS connection failure, err: %s", err.Error())
		return
	}

	ctx := ddsvc.ProcessArgs(svcName)
	if ctx == nil {
		return
	}

	if !data.InitLocalDatabase(ctx) {
		panic("Critical internal error")
	}

	routes.RegisterRoutes()
	go ddnats.SendHeartbeat(svcName)
	ddsvc.RunService(svcName, runEngine)

	log.Printf("Exiting ...")
}

func runEngine() {
	logger.Info("Microservices", "OPC DA microservice running")
}
