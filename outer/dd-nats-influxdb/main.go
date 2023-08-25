package main

import (
	"dd-nats/common/ddnats"
	"dd-nats/common/ddsvc"
	"dd-nats/common/logger"
	"log"
	"strconv"

	"github.com/nats-io/nats.go"
)

var emitter InfluxDBEmitter
var svc *ddsvc.DdUsvc
var sub *nats.Subscription

func main() {
	if svc = ddsvc.InitService("dd-nats-influxdb"); svc != nil {
		svc.RunService(runEngine)
	}

	log.Printf("Exiting ...")
}

func runEngine(svc *ddsvc.DdUsvc) {
	logger.Info("Microservices", "InfluxDB microservice running")
	registerRoutes()

	emitter.Host = svc.Get("host", "localhost")
	emitter.Database = svc.Get("database", "process")
	emitter.Port, _ = strconv.Atoi(svc.Get("port", "8086"))
	emitter.Batchsize, _ = strconv.Atoi(svc.Get("batchsize", "5"))
	emitter.InitEmitter()

	topic := svc.Get("topic", "inner.process.actual")
	sub, _ = ddnats.Subscribe(topic, emitter.ProcessDataPointHandler)
	ddnats.Subscribe("usvc.ddnatsinfluxdb.event.settingschanged", settingsChangedHandler)
}

func settingsChangedHandler(nmsg *nats.Msg) {
	sub.Unsubscribe()
	topic := svc.Get("topic", "inner.process.actual")
	sub, _ = ddnats.Subscribe(topic, emitter.ProcessDataPointHandler)
}
