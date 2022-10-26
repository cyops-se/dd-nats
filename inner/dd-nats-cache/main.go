package main

import (
	"dd-nats/common/ddsvc"
	"dd-nats/inner/dd-nats-cache/app"
	"dd-nats/inner/dd-nats-cache/routes"
	"log"
)

func main() {
	if svc := ddsvc.InitService("dd-nats-cache"); svc != nil {
		if app.Init(svc) {
			routes.RegisterRoutes()
			svc.RunService(app.RunApp)
		}
	}

	log.Printf("Exiting ...")
}
