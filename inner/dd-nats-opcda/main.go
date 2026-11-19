package main

import (
	"dd-nats/common/ddsvc"
	"dd-nats/inner/dd-nats-opcda/app"
	"dd-nats/inner/dd-nats-opcda/routes"
	"log"
)

func main() {
	if svc := ddsvc.InitService("dd-nats-opcda"); svc != nil {
		if app.Init(svc) {
			routes.RegisterRoutes(svc)
			svc.RunService(app.RunApp)
		}
	}

	log.Printf("Exiting ...")
}
