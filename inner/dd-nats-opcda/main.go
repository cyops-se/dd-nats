package main

import (
	"dd-nats/common/ddsvc"
	"dd-nats/inner/dd-nats-opcda/app"
	"dd-nats/inner/dd-nats-opcda/routes"
	"log"
	"net"

	"github.com/nats-io/nats.go"
)

var forwarder chan *nats.Msg = make(chan *nats.Msg, 2000)
var udpconn net.Conn

var packet []byte

func main() {
	svc := ddsvc.InitService("dd-nats-opcda")

	if app.Init(svc) {
		routes.RegisterRoutes(svc)
		svc.RunService(app.RunApp)
	}

	log.Printf("Exiting ...")
}

// func main() {
// 	svcName := "dd-nats-opcda"
// 	_, err := ddnats.Connect(nats.DefaultURL)
// 	if err != nil {
// 		log.Printf("Exiting application due to NATS connection failure, err: %s", err.Error())
// 		return
// 	}

// 	ctx := ddsvc.ProcessArgs(svcName)
// 	if ctx == nil {
// 		return
// 	}

// 	if app.Init(ctx) {
// 		routes.RegisterRoutes()
// 		go ddnats.SendHeartbeat(ctx.Name)
// 		ddsvc.RunService(ctx.Name, app.RunApp)
// 	}

// 	log.Printf("Exiting ...")
// }
