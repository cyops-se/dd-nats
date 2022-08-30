package main

import (
	"dd-nats/common/ddnats"
	"dd-nats/common/ddsvc"
	"dd-nats/inner/dd-nats-opcda/routes"
	"log"
	"net"
	"os"

	"github.com/nats-io/nats.go"
)

var forwarder chan *nats.Msg = make(chan *nats.Msg, 2000)
var udpconn net.Conn

var packet []byte

func main() {
	svcName := "dd-nats-opcda"
	nc, err := ddnats.Connect(nats.DefaultURL)
	if err != nil {
		log.Printf("Exiting application due to NATS connection failure, err: %s", err.Error())
		return
	}

	if ctx := ddsvc.ProcessArgs(svcName); ctx == nil {
		return
	}

	routes.RegisterRoutes(nc)
	go ddnats.SendHeartbeat(os.Args[0], nc)
	ddsvc.RunService(svcName, runEngine)

	log.Printf("Exiting ...")
}

func runEngine() {
	log.Println("Engine running ...")
}
