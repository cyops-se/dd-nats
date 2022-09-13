package ddnats

import (
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
)

func Publish(subject string, response interface{}) error {
	if lnc == nil {
		return fmt.Errorf("Failed to publish subject '%s': No connection to NATS", subject)
	}

	data, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("Failed to publish subject '%s': %s", subject, err.Error())
	}

	return lnc.Publish(subject, data)
}

func PublishError(f string, a ...interface{}) error {
	subject := "system.error"
	response := fmt.Sprintf(f, a...)
	return Publish(subject, response)
}

func Respond(msg *nats.Msg, response interface{}) error {
	if lnc == nil {
		return fmt.Errorf("Failed to respond to request with subject '%s': No connection to NATS", msg.Subject)
	}

	data, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("Failed to respond to request with subject '%s': %s", msg.Subject, err.Error())
	}

	// log.Println("nats.respond: ", string(data), response)
	return msg.Respond([]byte(data))
}

func Event(subject string, arg interface{}) error {
	return Publish("system.event."+subject, arg)
}
