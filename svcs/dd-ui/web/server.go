package web

import (
	"dd-nats/common/ddsvc"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"strconv"

	"dd-nats/svcs/dd-ui/routes"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/websocket/v2"
)

//go:embed static/index.html
var admin string

//go:embed static/*
var static embed.FS

// websocket connections
type WebSocketMessage struct {
	Topic   string      `json:"topic"`
	Message interface{} `json:"message"`
}

type WebSocketClient struct {
	connection *websocket.Conn
	close      chan string
}

var clients = make(map[*websocket.Conn]*WebSocketClient)
var register = make(chan *WebSocketClient, 5)
var broadcast = make(chan *WebSocketMessage, 100)

func handlePanic() {
	if r := recover(); r != nil {
		log.Printf("Servers panic, recovery: %#v", r)
	}
}

func RunWeb(usvc *ddsvc.DdUsvc) {
	defer handlePanic()

	go runSocketActions()

	// http.FS can be used to create a http Filesystem
	subFS2, err := fs.Sub(static, "static")
	if err != nil {
		usvc.Error("Web server", "Failed to load embedded static files: %s", err.Error())
		return
	}
	var staticFS = http.FS(subFS2)

	// Set a file transfer limit to 50MB
	app := fiber.New(fiber.Config{StrictRouting: true, BodyLimit: 50 * 1024 * 1024})
	if usvc.Context.Trace {
		app.Use(logger.New())
	}

	app.Use("/", filesystem.New(filesystem.Config{
		Root:   staticFS,
		Browse: false,
	}))

	app.Get("/ui/*", func(ctx *fiber.Ctx) error {
		ctx.Status(200)
		ctx.Set(fiber.HeaderContentType, fiber.MIMETextHTML)
		return ctx.SendString(admin)
	})

	app.Use("/static", filesystem.New(filesystem.Config{
		Root:   staticFS,
		Browse: false,
	}))

	// WebSocket registration
	app.Get("/ws", websocket.New(func(c *websocket.Conn) {
		client := &WebSocketClient{c, make(chan string, 1)}

		register <- client
		reason := <-client.close
		log.Println("websocket close signal received:", reason)
	}))

	api := app.Group("/api")
	routes.RegisterRoutes(api, usvc)

	usvc.Subscribe("system.heartbeat", func(topic string, responseTopic string, data []byte) error {
		// Never block the message broker callback, drop the message if the websocket clients are too slow
		select {
		case broadcast <- &WebSocketMessage{Topic: topic, Message: string(data)}:
		default:
		}
		return nil
	})

	usvc.OnStop(func() { app.Shutdown() })

	address := net.JoinHostPort(usvc.Context.BindIP, strconv.Itoa(usvc.Context.Port))
	usvc.Info("Web server", "Listening on %s", address)
	if err := app.Listen(address); err != nil {
		usvc.Error("Web server", "Failed to listen on %s: %s", address, err.Error())
	}
}

func runSocketActions() {
	for {
		select {
		case client := <-register:
			clients[client.connection] = client
			log.Println("new websocket connection registered")

		case msg := <-broadcast:
			for connection, client := range clients {
				if err := connection.WriteJSON(msg); err != nil {
					connection.WriteMessage(websocket.CloseMessage, []byte{})
					connection.Close()
					delete(clients, connection) // deleting during range is safe in Go
					client.close <- fmt.Sprintf("write failed: %s", err.Error())
					log.Println("websocket connection unregistered")
				}
			}
		}
	}
}
