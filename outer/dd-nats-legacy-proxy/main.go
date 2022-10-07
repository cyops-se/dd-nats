package main

import (
	"dd-nats/common/ddsvc"
	"log"
)

func main() {
	if svc := ddsvc.InitService("dd-nats-legacy-proxy"); svc != nil {
		svc.RunService(runEngine)
	}

	log.Printf("Exiting ...")
}

func runEngine(svc *ddsvc.DdUsvc) {
}
