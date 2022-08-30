package ddnats

import (
	"encoding/json"
	"fmt"
)

func Publish(subject string, response interface{}) error {
	if lnc == nil {
		return fmt.Errorf("Failed to publish subject '%s': No connection to NATS", subject)
	}

	data, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("Failed to publish subject '%s': %s", subject, err.Error())
	}

	return lnc.Publish(subject, []byte(data))
}

func PublishError(f string, a ...interface{}) error {
	subject := "system.error"
	response := fmt.Sprintf(f, a...)
	return Publish(subject, response)
}
