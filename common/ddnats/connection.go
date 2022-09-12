package ddnats

import (
	"dd-nats/common/types"
	"encoding/json"
	"log"
	"os"
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

	log.Printf("Connected ...")
	lnc = nc
	return nc, err
}

func SendHeartbeat(appname string) {
	ticker := time.NewTicker(1 * time.Second)
	hostname, _ := os.Hostname()

	for {
		<-ticker.C
		heartbeat := &types.Heartbeat{Hostname: hostname, AppName: appname, Version: "0.0.1"}
		payload, _ := json.Marshal(heartbeat)
		lnc.Publish("system.heartbeat", payload)
	}
}
