package main

import (
	"dd-nats/common/ddnats"
	"dd-nats/common/types"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
)

var forwarder chan *nats.Msg = make(chan *nats.Msg, 2000)
var udpconn net.Conn

var packet []byte

func main() {
	svcName := "dd-nats-inner-proxy"
	if err := connectUDP(); err != nil {
		log.Printf("Exiting application due to UDP connection failure, err: %s", err.Error())
		return
	}

	udpconn.SetWriteDeadline(time.Time{})

	nc, err := ddnats.Connect(nats.DefaultURL)
	if err != nil {
		log.Printf("Exiting application due to NATS connection failure, err: %s", err.Error())
		return
	}

	// Set up UDP sender
	go sendUDP()

	// Set up subscription wildcard for messages that should be forwarded to the outer proxy
	nc.Subscribe("forward.>", callbackHandler)

	// Set up subscription for system messages that should be forwarded to the outer proxy
	nc.Subscribe("system.>", callbackHandler)

	// Set up heartbeat
	go ddnats.SendHeartbeat(svcName)

	// Sleep until interrupted
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Printf("Exiting ...")
}

func connectUDP() (err error) {
	target := fmt.Sprintf("%s:%d", "localhost", 4359)
	udpconn, err = net.Dial("udp", target)
	return err
}

func callbackHandler(msg *nats.Msg) {
	// Use of channel to serialize NATS message callbacks
	forwarder <- msg
}

func sendUDP() {
	totmsgs := uint64(0)
	totpkts := uint64(0)
	counter := uint32(0)
	packet = make([]byte, 1200)

	for {
		msg := <-forwarder
		sdata := []byte("inner." + msg.Subject)

		copy(packet, []byte("$MAGIC8$"))
		binary.LittleEndian.PutUint32(packet[8:], counter)
		binary.LittleEndian.PutUint32(packet[12:], uint32(len(sdata)))
		binary.LittleEndian.PutUint32(packet[16:], uint32(len(msg.Data)))
		copy(packet[20:], sdata)
		udpconn.Write(packet)
		counter++
		totpkts++

		index := 0
		packetsize := cap(packet) - 8
		remainingsize := len(msg.Data)
		for remainingsize > 0 {
			if remainingsize < packetsize {
				packetsize = remainingsize
			}

			binary.LittleEndian.PutUint32(packet, uint32(counter))
			binary.LittleEndian.PutUint32(packet[4:], uint32(packetsize))
			copy(packet[8:], msg.Data[index:packetsize+index])
			udpconn.Write(packet)
			counter++
			totpkts++

			remainingsize -= packetsize
			index += packetsize
			if counter%25 == 0 {
				time.Sleep(1 * time.Millisecond)
			}
		}

		totmsgs++

		stats := &types.UdpStatistics{TotalMsg: totmsgs, TotalPkts: totpkts}
		ddnats.Publish("stats.nats.totmsgs", stats)
	}
}
