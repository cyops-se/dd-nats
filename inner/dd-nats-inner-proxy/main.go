package main

import (
	"dd-nats/common/ddsvc"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

type ForwardMsg struct {
	subject string
	data    []byte
}

// var forwarder chan *nats.Msg = make(chan *nats.Msg, 2000)
var forwarder chan ForwardMsg = make(chan ForwardMsg, 2000)
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
		svc.Subscribe(strings.TrimSpace(topic), callbackHandler)
	}

	// Sleep until interrupted
	select {}
}

func connectUDP(host string, port int) (err error) {
	target := fmt.Sprintf("%s:%d", host, port)
	udpconn, err = net.Dial("udp", target)
	return err
}

func callbackHandler(topic string, responseTopic string, data []byte) error {
	// Use of channel to serialize NATS message callbacks
	forwarder <- ForwardMsg{subject: topic, data: data}
	return nil
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
		sdata := []byte(msg.subject) // full subject payload
		ssize := len(sdata)          // subject size
		remainingsize := len(msg.data)
		totalsize := remainingsize
		// totalpackets := (totalsize + ssize) / (packetlen - 20) // overhead is 20 bytes
		totalpackets := totalsize / (packetlen - 20 - ssize) // overhead is 20 bytes + size of subject (ssize)
		if totalsize%packetlen != 0 {
			totalpackets++
		}

		packetno := 0
		index := 0 // current position in the full message payload buffer

		for packetno < totalpackets {
			binary.LittleEndian.PutUint32(packet, uint32(msgid))
			binary.LittleEndian.PutUint32(packet[4:], uint32(totalsize))
			binary.LittleEndian.PutUint32(packet[8:], uint32(totalpackets))
			binary.LittleEndian.PutUint32(packet[12:], uint32(packetno))
			binary.LittleEndian.PutUint32(packet[16:], uint32(ssize))

			// only first packet should have a subject
			if packetno == 0 || ssize > 0 {
				copy(packet[20:], sdata)
			}

			headersize := 4*5 + ssize // the packet header should be 5 x 32bits (4 bytes) plus length of subject (in bytes)

			// remaining space in the current packet to send
			remainingspace := packetlen - headersize - 4 // we have to include the payload size space

			// remaining size of the data to send
			payloadsize := remainingsize

			// if the amount of data to be sent is larger than the remaining space in the packet,
			// we split the remaining data into multiple payloads, filling this packet before sending it
			if payloadsize >= remainingspace {
				payloadsize = remainingspace
			}

			// put in the size of the packet payload after the subject
			binary.LittleEndian.PutUint32(packet[headersize:], uint32(payloadsize))

			// put in the actual data payload, after we make sure the sizes are within limits
			psize := headersize + 4 + payloadsize
			if psize <= packetlen {
				copy(packet[headersize+4:], msg.data[index:payloadsize+index])
				udpconn.Write(packet[:headersize+4+payloadsize])
			} else {
				log.Printf("ERROR: trying to stuff the packet with more than fits, psize: %d, headersize: %d, payloadsize: %d", psize, headersize, payloadsize)
			}

			ssize = 0 // signals that subject shouldn't be included in subsequent packets

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
