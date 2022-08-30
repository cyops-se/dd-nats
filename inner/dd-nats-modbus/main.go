package main

import (
	"dd-nats/common/ddnats"
	"dd-nats/common/types"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/nats-io/nats.go"
)

func main() {
	nc, err := ddnats.Connect(nats.DefaultURL)
	if err != nil {
		log.Printf("Exiting application due to NATS connection failure, err: %s", err.Error())
		return
	}

	go ddnats.SendHeartbeat(os.Args[0], nc)
	readModbus(nc)

	// Sleep until interrupted
	// c := make(chan os.Signal)
	// signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	// <-c

	log.Printf("Exiting ...")
}

func readModbus(nc *nats.Conn) {
	samples := types.DataPointSamples{}
	for n := 0; n < 10; n++ {
		sample := types.DataPointSample{Timestamp: time.Now().UTC(), Identity: fmt.Sprintf("test_%d.asdfaas.asdf.asdf.tkjkjönwertkjh.kjlwölkjwret", n), Value: 4.5}
		samples = append(samples, sample)
	}

	for i := 0; i < 100; i++ {
		for n := 0; n < 2500; n++ {
			data, _ := json.Marshal(samples)
			nc.Publish("forward.datapoints", data)
		}

		log.Printf("Iteration %d", i)
		time.Sleep(1 * time.Second)
	}
}
