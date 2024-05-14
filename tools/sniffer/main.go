package main

import (
	"dd-nats/common/ddmb"
	"flag"
	"log"
	"runtime"
)

func main() {
	subject := flag.String("s", "forward.>", "The subject for which you want to sniff messsages")
	url := flag.String("url", "nats://localhost:4222", "NATS server URL")
	flag.Parse()

	if mb := ddmb.NewMessageBroker(*url); mb != nil {
		if err := mb.Connect(*url); err != nil {
			log.Printf("Exiting application due to NATS connection failure, err: %s", err.Error())
			return
		}

		mb.Subscribe(*subject, msgHandler)
	}

	runtime.Goexit()
}

func msgHandler(topic string, responseTopic string, data []byte) error {
	log.Println(topic, responseTopic, string(data))
	return nil
}
