package routes

import (
	"dd-nats/common/ddnats"
	"dd-nats/inner/dd-nats-cache/app"
	"dd-nats/inner/dd-nats-cache/messages"

	"github.com/nats-io/nats.go"
)

func RegisterRoutes() {
	ddnats.Subscribe("usvc.cache.getall", getAllCaches)
}

func getAllCaches(nmsg *nats.Msg) {
	var response messages.CacheResponse
	response.Success = true
	response.Info = app.GetCacheInfo()
	ddnats.Respond(nmsg, response)
}
