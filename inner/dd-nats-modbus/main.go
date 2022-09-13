package main

import (
	"dd-nats/common/ddnats"
	"dd-nats/common/types"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

func main() {
	svcName := "dd-nats-modbus"
	_, err := ddnats.Connect(nats.DefaultURL)
	if err != nil {
		log.Printf("Exiting application due to NATS connection failure, err: %s", err.Error())
		return
	}

	go ddnats.SendHeartbeat(svcName)
	readModbus()

	log.Printf("Exiting ...")
}

func readModbus() {
	samples := types.DataPointSamples{}
	for n := 0; n < 10; n++ {
		sample := types.DataPointSample{Timestamp: time.Now().UTC(), Identity: fmt.Sprintf("test_%d.asdfaas.asdf.asdf.tkjkjönwertkjh.kjlwölkjwret", n), Value: 4.5}
		samples = append(samples, sample)
	}

	for i := 0; i < 100; i++ {
		for n := 0; n < 2500; n++ {
			ddnats.Publish("forward.process", samples)
		}

		log.Printf("Iteration %d", i)
		time.Sleep(1 * time.Second)
	}
}
