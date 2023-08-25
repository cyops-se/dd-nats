package messages

import (
	"dd-nats/common/types"
	"dd-nats/inner/dd-nats-cache/app"
)

type CacheResponse struct {
	types.StatusResponse
	Info app.CacheInfo `json:"info"`
}
