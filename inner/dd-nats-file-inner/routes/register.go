package routes

import (
	"github.com/nats-io/nats.go"
)

func RegisterRoutes(nc *nats.Conn) {
	registerFolderRoutes()
}
