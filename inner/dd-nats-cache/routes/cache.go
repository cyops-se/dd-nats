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
	app.RefreshCache()
	// var err error
	var response messages.CacheResponse
	response.Success = false
	response.StatusMessage = "Not yet implemented"
	// if response.Items, err = app.GetGroups(); err != nil {
	// 	response.Success = false
	// 	response.StatusMessage = err.Error()
	// }

	ddnats.Respond(nmsg, response)
}
