package ddnats

import (
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

var lnc *nats.Conn

func Connect(url string) (*nats.Conn, error) {
	nc, err := nats.Connect(url)

	for err != nil {
		time.Sleep(5 * time.Second)
		log.Printf("Failed to connect to NATS server, retrying every 5 seconds ...")
		nc, err = nats.Connect(url)
	}

	log.Printf("Connected to %s ...", url)
	lnc = nc
	return nc, err
}
