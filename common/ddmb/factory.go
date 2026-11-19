package ddmb

import (
	"fmt"
	"log"
	"runtime/debug"
	"strings"
	"time"
)

// Default timeout for request/response style messages
const RequestTimeout = 5 * time.Second

type IMessageHandler func(topic string, responsetopic string, data []byte) error

type IMessageBroker interface {
	Connect(connectionUrl string) error
	Disconnect() error
	Publish(topic string, data interface{}) error
	Request(topic string, data interface{}) ([]byte, error)
	Subscribe(topic string, callback IMessageHandler) error
	Unsubscribe(topic string) error
}

func NewMessageBroker(connectionUrl string) IMessageBroker {
	name := strings.ToLower(connectionUrl)
	var mb IMessageBroker

	if strings.HasPrefix(name, "nats") {
		mb = NewNatsBroker()
	} else if strings.HasPrefix(name, "mqtt") {
		mb = NewMqttBroker()
	}

	return mb
}

// dispatch invokes a message handler and makes sure that a panic in a single
// handler doesn't bring down the whole process
func dispatch(callback IMessageHandler, topic string, responseTopic string, data []byte) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in message handler for topic '%s': %v\n%s", topic, r, string(debug.Stack()))
		}
	}()

	if err := callback(topic, responseTopic, data); err != nil {
		log.Printf("Message handler for topic '%s' returned error: %s", topic, err.Error())
	}
}

// toPayload converts data to a byte slice, raw byte slices are passed as is,
// anything else is JSON encoded
func toPayload(topic string, data interface{}, marshal func(interface{}) ([]byte, error)) ([]byte, error) {
	if b, ok := data.([]byte); ok {
		return b, nil
	}

	payload, err := marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload for subject '%s': %s", topic, err.Error())
	}

	return payload, nil
}
