package main

import (
	"dd-nats/common/ddsvc"
	"log"
)

var svc *ddsvc.DdUsvc

func main() {
	if svc = ddsvc.InitService("dd-nats-process-filter"); svc != nil {
		svc.RunService(runEngine)
	}

	log.Printf("Exiting ...")
}

func runEngine(svc *ddsvc.DdUsvc) {
	svc.Info("Microservices", "Process filter microservice running")

	datapoints = make(map[string]*filteredPoint)
	loadFilterMeta()
	registerFilterRoutes()

	// Listen for incoming process data from the inside
	topic := svc.Get("topic", "inner.process.actual")
	svc.Subscribe(topic, processDataPointHandler)

	// Listen for changes to meta data
	svc.Subscribe("system.event.timescale.metaupdated", processMetaUpdate)
}
