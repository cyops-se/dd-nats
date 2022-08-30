package routes

import (
	"dd-nats/common/logger"

	"github.com/nats-io/nats.go"
)

var lnc *nats.Conn

func RegisterRoutes(nc *nats.Conn) {
	lnc = nc
	lnc.Subscribe("system.heartbeat", systemHeartbeats)
	registerGroupRoutes(lnc)
}

func systemHeartbeats(msg *nats.Msg) {
	logger.Trace("heartbeat received", "%s", string(msg.Data))
}
