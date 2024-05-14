package routes

import (
	"dd-nats/common/ddsvc"

	"github.com/gofiber/fiber/v2"
)

var usvc *ddsvc.DdUsvc

func RegisterRoutes(api fiber.Router, svc *ddsvc.DdUsvc) {
	usvc = svc
	registerFileTransferRoutes(api)
	registerNatsRoutes(api)
	registerSystemRoutes(api)
}
