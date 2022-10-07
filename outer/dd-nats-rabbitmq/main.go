package main

import (
	"dd-nats/common/ddnats"
	"dd-nats/common/ddsvc"
	"dd-nats/common/logger"
	"log"
)

var emitter RabbitMQEmitter

func main() {
	if svc := ddsvc.InitService("dd-nats-rabbitmq"); svc != nil {
		svc.RunService(runEngine)
	}

	log.Printf("Exiting ...")
}

func runEngine(svc *ddsvc.DdUsvc) {
	logger.Info("Microservices", "RabbitMQ microservice running")
	emitter.ChannelName = "hello"
	emitter.Durable = true
	emitter.Urls = []string{"amqp://admin:hemligt@192.168.1.65:5672/"}
	emitter.InitEmitter()

	// Listen for incoming process data
	ddnats.Subscribe("inner.process.actual", emitter.processDataPointHandler)
}
