package ddnats

import (
	"fmt"

	"github.com/nats-io/nats.go"
)

func Subscribe(subject string, cb nats.MsgHandler) (*nats.Subscription, error) {
	if lnc == nil {
		return nil, fmt.Errorf("Failed to subscribe to subject '%s': No connection to NATS", subject)
	}

	sub, err := lnc.Subscribe(subject, cb)
	if err != nil {
		return nil, fmt.Errorf("Failed to subscribe to subject '%s': %s", subject, err.Error())
	}

	return sub, nil
}
