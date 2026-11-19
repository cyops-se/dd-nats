package main

import "dd-nats/common/types"

func registerRoutes() {
	svc.Subscribe("usvc.influxdb.ping", ping)
}

func ping(subject string, responseTopic string, data []byte) error {
	return svc.Publish(responseTopic, types.StatusResponse{Success: true})
}
