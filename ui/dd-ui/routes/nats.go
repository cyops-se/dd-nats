package routes

import (
	"dd-nats/common/ddnats"
	"dd-nats/common/logger"

	"github.com/gofiber/fiber/v2"
)

type natsMsg struct {
	Subject string                 `json:"subject"`
	Payload map[string]interface{} `json:"payload"` // JSON encoded string
}

func RegisterNatsRoutes(api fiber.Router) {
	api.Post("/nats/request", RequestNats)
}

func RequestNats(c *fiber.Ctx) error {
	var webrequest natsMsg

	if err := c.BodyParser(&webrequest); err != nil {
		logger.Error("NATS request failed", "Failed to map provided data to natsMsg: %s", err.Error())
		return c.Status(503).SendString(err.Error())
	}

	if reply, err := ddnats.Request(webrequest.Subject, webrequest.Payload); err == nil {
		return c.Status(fiber.StatusOK).SendString(string(reply.Data)) // ddnats.Respond() already serialized the response to a JSON string
	} else {
		logger.Error("NATS request failed", "request: %s, failed: %s", webrequest.Subject, err.Error())
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}
}
