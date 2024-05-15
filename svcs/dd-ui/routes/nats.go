package routes

import (
	"github.com/gofiber/fiber/v2"
)

type natsMsg struct {
	Subject string                 `json:"subject"`
	Payload map[string]interface{} `json:"payload"` // JSON encoded string
}

func registerNatsRoutes(api fiber.Router) {
	api.Post("/nats/request", RequestMessageBroker)
}

func RequestMessageBroker(c *fiber.Ctx) error {
	var webrequest natsMsg

	if err := c.BodyParser(&webrequest); err != nil {
		usvc.Error("Message broker request failed", "Failed to map provided data to natsMsg: %s", err.Error())
		return c.Status(503).SendString(err.Error())
	}

	if data, err := usvc.Request(webrequest.Subject, webrequest.Payload); err == nil {
		return c.Status(fiber.StatusOK).SendString(string(data)) // ddnats.Respond() already serialized the response to a JSON string
	} else {
		usvc.Error("Message broker request failed", "request: %s, failed: %s", webrequest.Subject, err.Error())
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}
}
