package routes

import (
	"dd-nats/common/ddnats"
	"dd-nats/common/ddsvc"

	"github.com/nats-io/nats.go"
)

func registerSystemRoutes(svc *ddsvc.DdUsvc) {
	ddnats.Subscribe("system.heartbeat", systemHeartbeats)
}

func systemHeartbeats(msg *nats.Msg) {
	// logger.Trace("heartbeat received", "%s", string(msg.Data))
}
