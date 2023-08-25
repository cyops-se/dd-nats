package main

import (
	"dd-nats/common/ddnats"

	"github.com/nats-io/nats.go"
)

func registerRoutes() {
	ddnats.Subscribe("usvc.influxdb.ping", ping)
}

func ping(nmsg *nats.Msg) {
	ddnats.Respond(nmsg, nil)
}
