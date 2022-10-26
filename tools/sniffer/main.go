package main

import (
	"dd-nats/common/ddnats"
	"flag"
	"log"
	"runtime"

	"github.com/nats-io/nats.go"
)

func main() {
	subject := flag.String("s", "forward.>", "The subject for which you want to sniff messsages")
	url := flag.String("url", "nats://localhost:4222", "NATS server URL")
	flag.Parse()

	nc, err := ddnats.Connect(*url)
	if err != nil {
		log.Printf("Exiting application due to NATS connection failure, err: %s", err.Error())
		return
	}

	nc.Subscribe(*subject, msgHandler)
	runtime.Goexit()
}

func msgHandler(msg *nats.Msg) {
	log.Println(msg.Subject, string(msg.Data))
}
