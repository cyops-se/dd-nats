package main

import (
	"dd-nats/common/ddnats"
	"dd-nats/common/ddsvc"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
)

var forwarder chan *nats.Msg = make(chan *nats.Msg, 2000)
var udpconn net.Conn

var packet []byte

func main() {
	if svc := ddsvc.InitService("dd-nats-inner-proxy"); svc != nil {
		svc.RunService(runService)
	}

	log.Printf("Exiting ...")
}

func runService(svc *ddsvc.DdUsvc) {

	host := svc.Get("proxy-ip", "192.168.2.101")
	port := 4359
	if err := connectUDP(host, port); err != nil {
		log.Printf("Exiting application due to UDP connection failure, err: %s", err.Error())
		return
	}

	udpconn.SetWriteDeadline(time.Time{})

	// Set up UDP sender
	go sendUDP()

	topicstr := svc.Get("topics", "process.>,file.>,system.log.>,system.heartbeat")
	topics := strings.Split(topicstr, ",")

	for _, topic := range topics {
		ddnats.Subscribe(strings.TrimSpace(topic), callbackHandler)
	}

	// Sleep until interrupted
	select {}
}

func connectUDP(host string, port int) (err error) {
	target := fmt.Sprintf("%s:%d", host, port)
	udpconn, err = net.Dial("udp", target)
	return err
}

func callbackHandler(msg *nats.Msg) {
	// Use of channel to serialize NATS message callbacks
	forwarder <- msg
}

// packet layout
// | uint32 | uint32 | uint32 | uint32 | uint32 | [subjectsize]byte | uint32 | [packetsize]byte    |
// |--------|--------|--------|--------|--------|-------------------|--------|---------------------|
// | msgid  | total  | total  | packet | subject| subject...        | payload| payload fragment... |
// |        | size   | packets| no     | size   | (packetno = 0)    | size   |                     |

func sendUDP() {
	msgid := uint32(0)
	packetlen := 1480
	packet = make([]byte, packetlen)
	packetcount := uint64(0)
	interval := uint64(20) // number of packets before a break
	delay := 50            // in millisec

	for {
		msg := <-forwarder
		msgid++
		sdata := []byte(msg.Subject) // full subject payload
		ssize := len(sdata)
		remainingsize := len(msg.Data)

		totalsize := len(msg.Data)
		totalpackets := (totalsize + ssize) / (packetlen - 24) // overhead is 24 bytes
		if totalsize%packetlen != 0 {
			totalpackets++
		}

		packetno := 0
		index := 0 // current position in the full message payload buffer

		for packetno < totalpackets {
			headersize := 4*5 + ssize
			binary.LittleEndian.PutUint32(packet, uint32(msgid))
			binary.LittleEndian.PutUint32(packet[4:], uint32(totalsize))
			binary.LittleEndian.PutUint32(packet[8:], uint32(totalpackets))
			binary.LittleEndian.PutUint32(packet[12:], uint32(packetno))
			binary.LittleEndian.PutUint32(packet[16:], uint32(ssize))
			copy(packet[20:], sdata)

			remainingspace := packetlen - headersize
			payloadsize := remainingsize

			if payloadsize > remainingspace {
				payloadsize = remainingspace - 4 // include the payload size slot
			}

			binary.LittleEndian.PutUint32(packet[headersize:], uint32(payloadsize))
			copy(packet[headersize+4:], msg.Data[index:payloadsize+index])
			udpconn.Write(packet[:headersize+4+payloadsize])
			ssize = 0

			remainingsize -= payloadsize
			index += payloadsize
			packetno++

			packetcount++
			if packetcount%interval == 0 {
				time.Sleep(time.Duration(delay) * time.Millisecond)
			}
		}
	}
}
