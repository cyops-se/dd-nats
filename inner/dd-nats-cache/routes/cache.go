package routes

import (
	"dd-nats/common/ddsvc"
	"dd-nats/common/types"
	"dd-nats/inner/dd-nats-cache/app"
	"dd-nats/inner/dd-nats-cache/messages"
	"encoding/json"
)

var usvc *ddsvc.DdUsvc

func RegisterRoutes(svc *ddsvc.DdUsvc) {
	usvc = svc
	usvc.Subscribe("usvc.cache.getall", getAllCaches)
	usvc.Subscribe("usvc.cache.resend", resendCacheItems)
}

func getAllCaches(subject string, responseTopic string, data []byte) error {
	var response messages.CacheResponse
	response.Success = true
	response.Info = app.GetCacheInfo()
	return usvc.Publish(responseTopic, response)
}

func resendCacheItems(subject string, responseTopic string, data []byte) error {
	response := &types.StatusResponse{Success: true}

	var items []app.CacheItem
	if err := json.Unmarshal(data, &items); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	} else {
		// count := app.ResendCacheItems(items)
		app.ResendCacheItems(items)
	}

	return usvc.Publish(responseTopic, response)
}
