package routes

import (
	"dd-nats/common/ddnats"

	"github.com/nats-io/nats.go"
)

func registerSystemRoutes() {
	ddnats.Subscribe("system.heartbeat", systemHeartbeats)
}

func systemHeartbeats(msg *nats.Msg) {
	// logger.Trace("heartbeat received", "%s", string(msg.Data))
}
