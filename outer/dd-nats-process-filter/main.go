package main

import (
	"dd-nats/common/ddsvc"
	"log"
)

var svc *ddsvc.DdUsvc
var inputTopic string

func main() {
	if svc = ddsvc.InitService("dd-nats-process-filter"); svc != nil {
		svc.RunService(runEngine)
	}

	log.Printf("Exiting ...")
}

func runEngine(s *ddsvc.DdUsvc) {
	svc.Info("Microservices", "Process filter microservice running")

	if err := loadFilterMeta(); err != nil {
		svc.Error("Process filter", "Failed to load filter data: %s", err.Error())
	}

	registerFilterRoutes()

	// Listen for incoming process data from the inside
	inputTopic = svc.Get("topic", "inner.process.actual")
	svc.Subscribe(inputTopic, processDataPointHandler)
	go processPeriodSubmissionTask()

	// Listen for changes to meta data
	svc.Subscribe("system.event.timescale.metaupdated", processMetaUpdate)
}
