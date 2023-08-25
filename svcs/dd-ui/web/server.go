package web

import (
	"dd-nats/common/ddnats"
	"dd-nats/common/types"
	"dd-nats/svcs/dd-ui/routes"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/websocket/v2"
	"github.com/nats-io/nats.go"
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

var dropList []int
var ws []*websocket.Conn
var wsMutex sync.Mutex
var clients = make(map[*websocket.Conn]*WebSocketClient)
var register = make(chan *WebSocketClient, 5)
var unregister = make(chan *websocket.Conn, 50)
var broadcast = make(chan *WebSocketMessage, 5)

func handlePanic() {
	if r := recover(); r != nil {
		log.Printf("Servers panic, recovery: %#v", r)
		return
	}
}

func RunWeb(args types.Context) {
	defer handlePanic()

	go runSocketActions()

	// http.FS can be used to create a http Filesystem
	subFS2, _ := fs.Sub(static, "static")
	var staticFS = http.FS(subFS2)

	// Set a file transfer limit to 50MB
	app := fiber.New(fiber.Config{StrictRouting: true, BodyLimit: 50 * 1024 * 1024})
	if args.Trace {
		app.Use(logger.New())
	}

	app.Use("/", filesystem.New(filesystem.Config{
		Root:   staticFS,
		Browse: false,
	}))

	app.Get("/ui/*", func(ctx *fiber.Ctx) error {
		ctx.Status(200)
		ctx.Set(fiber.HeaderContentType, fiber.MIMETextHTML)
		// return ctx.Send([]byte(admin))
		return ctx.SendString(admin)
	})

	app.Use("/static", filesystem.New(filesystem.Config{
		Root:   staticFS,
		Browse: false,
	}))

	// WebSocket registration
	app.Get("/ws", websocket.New(func(c *websocket.Conn) {
		client := &WebSocketClient{c, make(chan string)}

		register <- client
		close := <-client.close
		log.Println("websocket close signal recevied:", close)
	}))

	api := app.Group("/api")
	routes.RegisterSystemRoutes(api)
	routes.RegisterNatsRoutes(api)
	routes.RegisterFileTransferRoutes(api)

	ddnats.Subscribe(">", func(m *nats.Msg) {
		broadcast <- &WebSocketMessage{Topic: m.Subject, Message: string(m.Data)}
	})

	app.Listen(fmt.Sprintf(":%d", args.Port))

	select {}
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
					client.close <- "timetoexit"
					unregister <- connection
					connection.WriteMessage(websocket.CloseMessage, []byte{})
					connection.Close()
				}
			}

		case connection := <-unregister:
			delete(clients, connection)
			log.Println("websocket connection unregistered")
		}
	}
}
