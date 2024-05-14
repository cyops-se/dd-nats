package main

import (
	"dd-nats/common/ddsvc"
	"log"
	"strconv"
)

var emitter TimescaleEmitter
var svc *ddsvc.DdUsvc

func main() {
	if svc = ddsvc.InitService("dd-nats-timescale"); svc != nil {
		svc.RunService(runEngine)
	}

	log.Printf("Exiting ...")
}

func runEngine(svc *ddsvc.DdUsvc) {
	svc.Info("Microservices", "Timescale microservice running")
	registerRoutes()

	emitter.Host = svc.Get("host", "localhost")
	emitter.Database = svc.Get("database", "postgres")
	emitter.Port, _ = strconv.Atoi(svc.Get("port", "5432"))
	emitter.User = svc.Get("user", "postgres")
	emitter.Password = svc.Get("password", "")
	emitter.Batchsize, _ = strconv.Atoi(svc.Get("batchsize", "5"))
	emitter.InitEmitter()

	topic := svc.Get("topic", "inner.process.actual")
	svc.Subscribe(topic, emitter.ProcessDataPointHandler)
	svc.Subscribe("usvc.ddnatstimescale.event.settingschanged", settingsChangedHandler)
}

func settingsChangedHandler(subject string, responseTopic string, data []byte) error {
	topic := svc.Get("topic", "inner.process.actual")
	return svc.Subscribe(topic, emitter.ProcessDataPointHandler)
}
