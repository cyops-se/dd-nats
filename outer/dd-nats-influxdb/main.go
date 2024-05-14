package main

import (
	"dd-nats/common/ddsvc"
	"log"
	"strconv"
)

var emitter InfluxDBEmitter
var svc *ddsvc.DdUsvc

func main() {
	if svc = ddsvc.InitService("dd-nats-influxdb"); svc != nil {
		svc.RunService(runEngine)
	}

	log.Printf("Exiting ...")
}

func runEngine(svc *ddsvc.DdUsvc) {
	svc.Info("Microservices", "InfluxDB microservice running")
	registerRoutes()

	emitter.Host = svc.Get("host", "localhost")
	emitter.Database = svc.Get("database", "process")
	emitter.Port, _ = strconv.Atoi(svc.Get("port", "8086"))
	emitter.Batchsize, _ = strconv.Atoi(svc.Get("batchsize", "5"))
	emitter.InitEmitter(svc)

	topic := svc.Get("topic", "inner.process.actual")
	svc.Subscribe(topic, emitter.ProcessDataPointHandler)
	svc.Subscribe("usvc.ddnatsinfluxdb.event.settingschanged", settingsChangedHandler)
}

func settingsChangedHandler(subject string, responseTopic string, data []byte) error {
	topic := svc.Get("topic", "inner.process.actual")
	svc.Subscribe(topic, emitter.ProcessDataPointHandler)
	return nil
}
