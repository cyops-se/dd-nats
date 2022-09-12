package ddnats

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

func Request(subject string, payload interface{}) (*nats.Msg, error) {
	if lnc == nil {
		return nil, fmt.Errorf("Failed to request subject '%s': No connection to NATS", subject)
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("Failed to request subject '%s': %s", subject, err.Error())
	}

	return lnc.Request(subject, []byte(data), 2*time.Second)
}
