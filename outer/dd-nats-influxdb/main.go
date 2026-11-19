package main

import (
	"dd-nats/common/ddsvc"
	"log"
	"sync"
)

var emitter InfluxDBEmitter
var svc *ddsvc.DdUsvc

var topicLock sync.Mutex
var currentTopic string

func main() {
	if svc = ddsvc.InitService("dd-nats-influxdb"); svc != nil {
		svc.RunService(runEngine)
	}

	log.Printf("Exiting ...")
}

func runEngine(s *ddsvc.DdUsvc) {
	svc.Info("Microservices", "InfluxDB microservice running")
	registerRoutes()

	emitter.Host = svc.Get("host", "localhost")
	emitter.Database = svc.Get("database", "process")
	emitter.Port = svc.GetInt("port", 8086)
	emitter.Batchsize = svc.GetInt("batchsize", 5)
	emitter.InitEmitter(svc)
	svc.OnStop(emitter.Stop)

	subscribeDataTopic()
	svc.Subscribe("usvc.ddnatsinfluxdb.event.settingschanged", settingsChangedHandler)
}

// (Re)subscribes the configured process data topic, making sure there is only
// ever one active subscription
func subscribeDataTopic() {
	topicLock.Lock()
	defer topicLock.Unlock()

	topic := svc.Get("topic", "inner.process.actual")
	if topic == currentTopic {
		return
	}

	if currentTopic != "" {
		svc.Unsubscribe(currentTopic)
	}

	if err := svc.Subscribe(topic, emitter.ProcessDataPointHandler); err == nil {
		currentTopic = topic
	}
}

func settingsChangedHandler(subject string, responseTopic string, data []byte) error {
	subscribeDataTopic()
	return nil
}
