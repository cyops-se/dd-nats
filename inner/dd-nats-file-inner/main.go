package main

import (
	"dd-nats/common/ddnats"
	"dd-nats/common/ddsvc"
	"dd-nats/inner/dd-nats-file-inner/app"
	"dd-nats/inner/dd-nats-file-inner/routes"
	"log"
	"net"

	"github.com/nats-io/nats.go"
)

var forwarder chan *nats.Msg = make(chan *nats.Msg, 2000)
var udpconn net.Conn
var err error

var packet []byte

func main() {
	svcName := "dd-nats-file-inner"
	_, err = ddnats.Connect(nats.DefaultURL)
	if err != nil {
		log.Printf("Exiting application due to NATS connection failure, err: %s", err.Error())
		return
	}

	ctx := ddsvc.ProcessArgs(svcName)
	if ctx == nil {
		return
	}

	routes.RegisterRoutes()
	go ddnats.SendHeartbeat(ctx.Name)
	ddsvc.RunService(ctx.Name, app.RunEngine)

	log.Printf("Exiting ...")
}
