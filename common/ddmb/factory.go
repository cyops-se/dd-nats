package ddmb

import "strings"

type IMessageHandler func(topic string, responsetopic string, data []byte) error

type IMessageBroker interface {
	Connect(connectionUrl string) error
	Disconnect() error
	Publish(topic string, data interface{}) error
	Request(topic string, data interface{}) ([]byte, error)
	Subscribe(topic string, callback IMessageHandler) error
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
