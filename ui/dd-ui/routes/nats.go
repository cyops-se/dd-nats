package routes

import (
	"dd-nats/ui/dd-ui/logger"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nats-io/nats.go"
)

type natsMsg struct {
	Subject string `json:"subject"`
	Payload string `json:"payload"` // JSON encoded string
}

var lnc *nats.Conn

func RegisterNatsRoutes(api fiber.Router, nc *nats.Conn) {
	lnc = nc
	api.Post("/nats/request", RequestNats)
}

func RequestNats(c *fiber.Ctx) error {
	var webrequest natsMsg

	if err := c.BodyParser(&webrequest); err != nil {
		logger.Log("error", "Failed to map provided data to natsMsg", err.Error())
		return c.Status(503).SendString(err.Error())
	}

	reply, _ := lnc.Request(webrequest.Subject, []byte(webrequest.Payload), time.Second*2)
	webreply := &natsMsg{Subject: reply.Subject, Payload: string(reply.Data)}

	return c.Status(fiber.StatusOK).JSON(webreply)
}
