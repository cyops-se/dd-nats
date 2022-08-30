package ddnats

import (
	"fmt"

	"github.com/nats-io/nats.go"
)

func Subscribe(subject string, cb nats.MsgHandler) error {
	if lnc == nil {
		return fmt.Errorf("Failed to subscribe to subject '%s': No connection to NATS", subject)
	}

	_, err := lnc.Subscribe(subject, cb)
	if err != nil {
		return fmt.Errorf("Failed to subscribe to subject '%s': %s", subject, err.Error())
	}

	return nil
}
