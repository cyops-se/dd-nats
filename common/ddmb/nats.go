package ddmb

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

type NatsMessageBroker struct {
	Error   error
	NatsCon *nats.Conn
	subs    map[string][]*nats.Subscription
	lock    sync.Mutex
}

func NewNatsBroker() IMessageBroker {
	mb := new(NatsMessageBroker)
	mb.subs = make(map[string][]*nats.Subscription)
	return mb
}

func (mb *NatsMessageBroker) Connect(connectionUrl string) error {
	options := []nats.Option{
		nats.MaxReconnects(-1), // never give up reconnecting (default is 60 attempts)
		nats.ReconnectWait(2 * time.Second),
		nats.DisconnectErrHandler(func(c *nats.Conn, err error) {
			if err != nil {
				log.Printf("Disconnected from NATS server: %s", err.Error())
			} else {
				log.Printf("Disconnected from NATS server")
			}
		}),
		nats.ReconnectHandler(func(c *nats.Conn) {
			log.Printf("Reconnected to NATS server: %s", c.ConnectedUrl())
		}),
		nats.ErrorHandler(func(c *nats.Conn, s *nats.Subscription, err error) {
			subject := ""
			if s != nil {
				subject = s.Subject
			}
			log.Printf("NATS error (subject '%s'): %s", subject, err.Error())
		}),
	}

	for {
		mb.NatsCon, mb.Error = nats.Connect(connectionUrl, options...)
		if mb.Error == nil {
			break
		}

		log.Printf("Failed to connect to NATS server, retrying in 5 seconds, error: %s", mb.Error.Error())
		time.Sleep(5 * time.Second)
	}

	log.Printf("Connected to %s", connectionUrl)
	return nil
}

func (mb *NatsMessageBroker) Disconnect() error {
	if mb.NatsCon == nil {
		return nil
	}

	err := mb.NatsCon.Drain()
	if err != nil {
		mb.NatsCon.Close()
	}

	return err
}

func (mb *NatsMessageBroker) Publish(topic string, data interface{}) error {
	if mb.NatsCon == nil {
		return fmt.Errorf("failed to publish subject '%s': No connection to NATS", topic)
	}

	payload, err := toPayload(topic, data, json.Marshal)
	if err != nil {
		return err
	}

	return mb.NatsCon.Publish(topic, payload)
}

func (mb *NatsMessageBroker) Request(topic string, data interface{}) ([]byte, error) {
	if mb.NatsCon == nil {
		return nil, fmt.Errorf("failed to publish subject '%s': No connection to NATS", topic)
	}

	payload, err := toPayload(topic, data, json.Marshal)
	if err != nil {
		return nil, err
	}

	msg, err := mb.NatsCon.Request(topic, payload, RequestTimeout)
	if err != nil {
		return nil, err
	}

	return msg.Data, nil
}

func (mb *NatsMessageBroker) Subscribe(topic string, callback IMessageHandler) error {
	if mb.NatsCon == nil {
		return fmt.Errorf("failed to subscribe to subject '%s': No connection to NATS", topic)
	}

	sub, err := mb.NatsCon.Subscribe(topic, func(msg *nats.Msg) {
		if msg == nil {
			log.Printf("Local NATS subscription dispatch received an empty message")
			return
		}

		dispatch(callback, msg.Subject, msg.Reply, msg.Data)
	})

	if err != nil {
		return fmt.Errorf("failed to subscribe to subject '%s': %s", topic, err.Error())
	}

	mb.lock.Lock()
	mb.subs[topic] = append(mb.subs[topic], sub)
	mb.lock.Unlock()

	return nil
}

func (mb *NatsMessageBroker) Unsubscribe(topic string) error {
	mb.lock.Lock()
	subs := mb.subs[topic]
	delete(mb.subs, topic)
	mb.lock.Unlock()

	var lasterr error
	for _, sub := range subs {
		if err := sub.Unsubscribe(); err != nil {
			lasterr = err
		}
	}

	return lasterr
}
