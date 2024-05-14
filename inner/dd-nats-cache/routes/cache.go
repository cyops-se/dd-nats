package routes

import (
	"dd-nats/common/ddsvc"
	"dd-nats/inner/dd-nats-cache/app"
	"dd-nats/inner/dd-nats-cache/messages"
)

var usvc *ddsvc.DdUsvc

func RegisterRoutes(svc *ddsvc.DdUsvc) {
	usvc = svc
	usvc.Subscribe("usvc.cache.getall", getAllCaches)
}

func getAllCaches(subject string, responseTopic string, data []byte) error {
	var response messages.CacheResponse
	response.Success = true
	response.Info = app.GetCacheInfo()
	return usvc.Publish(responseTopic, response)
}
