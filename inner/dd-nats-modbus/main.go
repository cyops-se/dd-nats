package main

import (
	"dd-nats/common/ddsvc"
	"dd-nats/inner/dd-nats-modbus/app"
	"dd-nats/inner/dd-nats-modbus/routes"
	"log"
)

func main() {
	if svc := ddsvc.InitService("dd-nats-modbus"); svc != nil {
		if app.Init(svc.Context) {
			routes.RegisterModbusSlaveRoutes()
			routes.RegisterModbusItemRoutes()
			svc.RunService(app.RunApp)
		}
	}

	log.Printf("Exiting ...")
}
