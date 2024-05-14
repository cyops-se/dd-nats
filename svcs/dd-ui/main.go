package main

import (
	"dd-nats/common/ddsvc"
	"dd-nats/common/types"
	"dd-nats/svcs/dd-ui/web"
	"log"
)

var ctx *types.Context

func main() {
	if svc := ddsvc.InitService("dd-ui"); svc != nil {
		svc.RunService(runEngine)
	}

	log.Printf("Exiting ...")
}

func runEngine(usvc *ddsvc.DdUsvc) {
	go web.RunWeb(usvc)
}
