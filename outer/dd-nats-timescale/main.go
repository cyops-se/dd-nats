package main

import (
	"dd-nats/common/ddnats"
	"dd-nats/common/ddsvc"
	"dd-nats/common/logger"
	"log"
	"strconv"

	"github.com/nats-io/nats.go"
)

var emitter TimescaleEmitter
var svc *ddsvc.DdUsvc
var sub *nats.Subscription

func main() {
	if svc = ddsvc.InitService("dd-nats-timescale"); svc != nil {
		svc.RunService(runEngine)
	}

	log.Printf("Exiting ...")
}

func runEngine(svc *ddsvc.DdUsvc) {
	logger.Info("Microservices", "Timescale microservice running")
	registerRoutes()

	emitter.Host = svc.Get("host", "localhost")
	emitter.Database = svc.Get("database", "postgres")
	emitter.Port, _ = strconv.Atoi(svc.Get("port", "5432"))
	emitter.User = svc.Get("user", "postgres")
	emitter.Password = svc.Get("password", "")
	emitter.Batchsize, _ = strconv.Atoi(svc.Get("batchsize", "5"))
	emitter.InitEmitter()

	topic := svc.Get("topic", "inner.process.actual")
	sub, _ = ddnats.Subscribe(topic, emitter.ProcessDataPointHandler)
	ddnats.Subscribe("usvc.ddnatstimescale.event.settingschanged", settingsChangedHandler)
}

func settingsChangedHandler(nmsg *nats.Msg) {
	sub.Unsubscribe()
	topic := svc.Get("topic", "inner.process.actual")
	sub, _ = ddnats.Subscribe(topic, emitter.ProcessDataPointHandler)
}
