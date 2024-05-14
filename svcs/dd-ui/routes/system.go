package routes

import (
	"dd-nats/common/ddsvc"

	"github.com/gofiber/fiber/v2"
)

func registerSystemRoutes(api fiber.Router) {
	api.Get("/system/sysinfo", GetSysInfo)
}

func GetSysInfo(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(ddsvc.SysInfo)
}
