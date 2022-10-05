package main

import (
	"dd-nats/common/ddsvc"
	"dd-nats/inner/dd-nats-file-inner/app"
	"dd-nats/inner/dd-nats-file-inner/routes"
	"log"
)

func main() {
	if svc := ddsvc.InitService("dd-nats-file-inner"); svc != nil {
		routes.RegisterRoutes()
		svc.RunService(app.RunEngine)
	}

	log.Printf("Exiting ...")
}
