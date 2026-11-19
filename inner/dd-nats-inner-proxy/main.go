package main

import (
	"dd-nats/common/ddsvc"
	"dd-nats/common/udpproto"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

type ForwardMsg struct {
	subject string
	data    []byte
}

var forwarder chan ForwardMsg = make(chan ForwardMsg, 2000)
var udpconn net.Conn

func main() {
	if svc := ddsvc.InitService("dd-nats-inner-proxy"); svc != nil {
		svc.RunService(runService)
	}

	log.Printf("Exiting ...")
}

func runService(svc *ddsvc.DdUsvc) {
	host := svc.Get("proxy-ip", "192.168.2.101")
	port := svc.GetInt("proxy-port", 4359)
	if err := connectUDP(host, port); err != nil {
		svc.Error("Inner proxy", "Exiting application due to UDP connection failure, err: %s", err.Error())
		return
	}

	svc.Trace("Inner proxy", "Successfully connected to: %s:%d", host, port)

	// Set up UDP sender
	go sendUDP(svc)

	topicstr := svc.Get("topics", "process.>,file.>,system.log.>,system.heartbeat")
	for _, topic := range strings.Split(topicstr, ",") {
		if topic = strings.TrimSpace(topic); topic != "" {
			svc.Subscribe(topic, callbackHandler)
		}
	}
}

func connectUDP(host string, port int) (err error) {
	target := net.JoinHostPort(host, strconv.Itoa(port))
	udpconn, err = net.Dial("udp4", target)
	return err
}

func callbackHandler(topic string, responseTopic string, data []byte) error {
	// Use of channel to serialize message callbacks
	forwarder <- ForwardMsg{subject: topic, data: data}
	return nil
}

func sendUDP(svc *ddsvc.DdUsvc) {
	msgid := uint32(0)
	packetlen := udpproto.DefaultPacketSize
	packetcount := uint64(0)
	interval := uint64(svc.GetInt("throttle-packets", 1000)) // number of packets before a break
	delay := time.Duration(svc.GetInt("throttle-delay-ms", 50)) * time.Millisecond
	if interval == 0 {
		interval = 1000
	}

	var lastWriteError time.Time

	for msg := range forwarder {
		msgid++
		err := udpproto.Fragment(msgid, msg.subject, msg.data, packetlen, func(packet []byte) error {
			if _, err := udpconn.Write(packet); err != nil {
				// Avoid flooding the log if the network is down
				if time.Since(lastWriteError) > 10*time.Second {
					svc.Error("Inner proxy", "Failed to write UDP packet: %s", err.Error())
					lastWriteError = time.Now()
				}
			}

			packetcount++
			if packetcount%interval == 0 {
				time.Sleep(delay)
			}

			return nil
		})

		if err != nil {
			svc.Error("Inner proxy", "Failed to send message, subject: %s, error: %s", msg.subject, err.Error())
		}
	}
}
