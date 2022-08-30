package logger

import (
	"log"
	"sync"

	"github.com/gofiber/websocket/v2"
)

// websocket connections
type WebSocketMessage struct {
	Topic   string      `json:"topic"`
	Message interface{} `json:"message"`
}

var dropList []int
var ws []*websocket.Conn
var wsMutex sync.Mutex

func RegisterWebsocket(c *websocket.Conn) {
	wsMutex.Lock()
	defer wsMutex.Unlock()
	ws = append(ws, c)
	log.Printf("Adding subscriber: %d", len(ws)-1)
	msg := &WebSocketMessage{Topic: "ws.meta", Message: "Subscription registered"}
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
