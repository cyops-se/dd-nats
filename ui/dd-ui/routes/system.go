package routes

import (
	"github.com/gofiber/fiber/v2"
)

type SystemInformation struct {
	GitVersion string `json:"gitversion"`
	GitCommit  string `json:"gitcommit"`
}

var SysInfo SystemInformation

func RegisterSystemRoutes(api fiber.Router) {
	api.Get("/system/sysinfo", GetSysInfo)
}

func GetSysInfo(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(SysInfo)
}
