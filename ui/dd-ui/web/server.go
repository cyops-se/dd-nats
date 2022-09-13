package web

import (
	"dd-nats/common/ddnats"
	"dd-nats/common/types"
	"dd-nats/ui/dd-ui/routes"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"sync"
	"time"

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

var dropList []int
var ws []*websocket.Conn
var wsMutex sync.Mutex

func handlePanic() {
	if r := recover(); r != nil {
		// ddlog.Error("RunWeb", "Panic, recovery: %#v", r)
		log.Printf("Servers panic, recovery: %#v", r)
		return
	}
}

func RunWeb(args types.Context) {
	defer handlePanic()

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
		RegisterWebsocket(c)
		for c.Conn != nil {
			time.Sleep(1)
		}
	}))

	api := app.Group("/api")
	routes.RegisterAuthRoutes(api)
	api.Get("/system/info", routes.GetSysInfo)

	// JWT Middleware
	// app.Use(jwtware.New(jwtware.Config{
	// 	SigningKey: []byte("897puihjöknawerthgfp7<yvalknp98h"),
	// }))

	routes.RegisterUserRoutes(api)
	routes.RegisterDataRoutes(api)
	routes.RegisterSystemRoutes(api)
	routes.RegisterNatsRoutes(api)
	routes.RegisterFileTransferRoutes(api)

	ddnats.Subscribe("stats.>", func(m *nats.Msg) {
		NotifySubscribers(m.Subject, string(m.Data))
	})

	ddnats.Subscribe("ui.>", func(m *nats.Msg) {
		NotifySubscribers(m.Subject, string(m.Data))
	})

	ddnats.Subscribe("system.>", func(m *nats.Msg) {
		NotifySubscribers(m.Subject, string(m.Data))
	})

	ddnats.Subscribe("inner.system.>", func(m *nats.Msg) {
		NotifySubscribers(m.Subject, string(m.Data))
	})

	app.Listen(":3000")

	select {}
}

func RegisterWebsocket(c *websocket.Conn) {
	wsMutex.Lock()
	defer wsMutex.Unlock()
	ws = append(ws, c)
	log.Printf("Adding subscriber: %d", len(ws)-1)
	msg := &WebSocketMessage{Topic: "ws.meta", Message: "{\"msg\": \"Subscription registered\"}"}
	c.WriteJSON(msg)
}

func NotifySubscribers(topic string, message interface{}) {
	wsMutex.Lock()
	defer wsMutex.Unlock()
	dropList = make([]int, 0)
	for i, c := range ws {
		if c == nil || c.Conn == nil {
			dropList = append(dropList, i)
			continue
		}

		if err := c.WriteJSON(&WebSocketMessage{Topic: topic, Message: message}); err != nil {
			// Remove connections that return an error
			dropList = append(dropList, i)
		}
	}
	dropSubscribers()
}

func dropSubscriber(i int) {
	log.Printf("Removing subscriber: %d", i)
	ws[i].Close()
	ws[i].Conn = nil
	ws[i] = ws[len(ws)-1]
	ws[len(ws)-1] = nil
	ws = ws[:len(ws)-1]
}

func dropSubscribers() {
	if len(ws) == 0 || len(dropList) == 0 {
		return
	}

	// Assume dropList is sorted in ascending order
	for i := len(dropList) - 1; i >= 0; i-- {
		dropSubscriber(dropList[i])
	}
}
