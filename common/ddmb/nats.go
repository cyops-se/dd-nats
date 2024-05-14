package ddmb

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

type NatsSubscription struct {
	name string
	cb   IMessageHandler
}

type NatsMessageBroker struct {
	Error   error
	NatsCon *nats.Conn
	subs    map[string]*NatsSubscription
}

func NewNatsBroker() IMessageBroker {
	mb := new(NatsMessageBroker)
	mb.subs = make(map[string]*NatsSubscription)
	return mb
}

func (mb *NatsMessageBroker) Connect(connectionUrl string) error {
	if mb.NatsCon, mb.Error = nats.Connect(connectionUrl); mb.Error != nil {
		for mb.Error != nil {
			log.Printf("Failed to connect to NATS server, retrying in 5 seconds, error: %s", mb.Error.Error())
			time.Sleep(5 * time.Second)
			mb.NatsCon, mb.Error = nats.Connect(connectionUrl)
		}
	}

	log.Printf("Connected to %s", connectionUrl)
	return nil
}

func (mb *NatsMessageBroker) Disconnect() error {
	return nil
}

func (mb *NatsMessageBroker) Publish(topic string, data interface{}) error {
	if mb.NatsCon == nil {
		return fmt.Errorf("failed to publish subject '%s': No connection to NATS", topic)
	}

	var payload []byte
	var err error

	if _, ok := data.([]byte); ok {
		payload = data.([]byte)
	} else {
		payload, err = json.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to publish subject '%s': %s", topic, err.Error())
		}
	}

	return mb.NatsCon.Publish(topic, payload)
}

func (mb *NatsMessageBroker) Request(topic string, data interface{}) ([]byte, error) {
	if mb.NatsCon == nil {
		return nil, fmt.Errorf("failed to publish subject '%s': No connection to NATS", topic)
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to publish subject '%s': %s", topic, err.Error())
	}

	msg, err := mb.NatsCon.Request(topic, payload, 2*time.Second)
	if err != nil {
		return nil, err
	}

	return msg.Data, err
}

func (mb *NatsMessageBroker) Subscribe(topic string, callback IMessageHandler) error {
	if mb.NatsCon == nil {
		return fmt.Errorf("failed to subscribe to subject '%s': No connection to NATS", topic)
	}

	sub := &NatsSubscription{name: topic, cb: callback}
	mb.NatsCon.Subscribe(topic, sub.localDispatch)

	return nil
}

func (sub *NatsSubscription) localDispatch(msg *nats.Msg) {
	if msg == nil {
		log.Printf("Local NATS subscription dispatch received an empty message")
		return
	}

	sub.cb(msg.Subject, msg.Reply, msg.Data)
}
