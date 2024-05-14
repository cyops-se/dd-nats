package main

import (
	"dd-nats/common/ddsvc"
	"log"
)

var svc *ddsvc.DdUsvc
var emitter RabbitMQEmitter

func main() {
	if svc = ddsvc.InitService("dd-nats-rabbitmq"); svc != nil {
		svc.RunService(runEngine)
	}

	log.Printf("Exiting ...")
}

func runEngine(svc *ddsvc.DdUsvc) {
	svc.Info("Microservices", "RabbitMQ microservice running")
	emitter.ChannelName = "hello"
	emitter.Durable = true
	emitter.Urls = []string{"amqp://user:password@127.0.0.1:5672/"}
	emitter.InitEmitter()

	// Listen for incoming process data
	svc.Subscribe("inner.process.actual", emitter.processDataPointHandler)
}
