package main

import (
	"dd-nats/common/ddsvc"
	"log"
	"strings"
)

var svc *ddsvc.DdUsvc
var emitter RabbitMQEmitter

func main() {
	if svc = ddsvc.InitService("dd-nats-rabbitmq"); svc != nil {
		svc.RunService(runEngine)
	}

	log.Printf("Exiting ...")
}

func runEngine(s *ddsvc.DdUsvc) {
	svc.Info("Microservices", "RabbitMQ microservice running")

	for _, u := range strings.Split(svc.Get("urls", "amqp://user:password@127.0.0.1:5672/"), ",") {
		if u = strings.TrimSpace(u); u != "" {
			emitter.Urls = append(emitter.Urls, u)
		}
	}

	emitter.ChannelName = svc.Get("queue", "hello")
	emitter.Durable = svc.Get("durable", "true") == "true"
	emitter.InitEmitter()

	// Listen for incoming process data
	svc.Subscribe(svc.Get("topic", "inner.process.actual"), emitter.processDataPointHandler)
}
