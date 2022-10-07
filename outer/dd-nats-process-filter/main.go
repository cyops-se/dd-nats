package main

import (
	"dd-nats/common/ddnats"
	"dd-nats/common/ddsvc"
	"dd-nats/common/logger"
	"log"
)

func main() {
	if svc := ddsvc.InitService("dd-nats-process-filter"); svc != nil {
		svc.RunService(runEngine)
	}

	log.Printf("Exiting ...")
}

func runEngine(svc *ddsvc.DdUsvc) {
	logger.Info("Microservices", "Process filter microservice running")

	datapoints = make(map[string]*filteredPoint)
	registerFilterRoutes()

	// Listen for incoming process data from the inside
	ddnats.Subscribe("inner.process.actual", processDataPointHandler)
}
