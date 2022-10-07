package main

import (
	"dd-nats/common/ddnats"
	"flag"
	"fmt"
	"log"
	"runtime"

	"github.com/nats-io/nats.go"
)

func main() {
	subject := flag.String("s", "forward.>", "The subject for which you want to sniff messsages")
	port := flag.Int("p", 4222, "NATS server port on localhost to use, for example if you are running two instances")
	flag.Parse()

	url := fmt.Sprintf("nats://localhost:%d", *port)
	nc, err := ddnats.Connect(url)
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
