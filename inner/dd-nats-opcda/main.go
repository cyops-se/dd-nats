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
	if svc := ddsvc.InitService("dd-nats-opcda"); svc != nil {
		if app.Init(svc) {
			routes.RegisterRoutes(svc)
			svc.RunService(app.RunApp)
		}
	}

	log.Printf("Exiting ...")
}
