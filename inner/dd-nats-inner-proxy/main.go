package main

import (
	"dd-nats/common/ddnats"
	"dd-nats/common/ddsvc"
	"dd-nats/common/types"
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

func sendUDP() {
	totmsgs := uint64(0)
	totpkts := uint64(0)
	counter := uint32(0)
	packet = make([]byte, 1200)

	for {
		msg := <-forwarder
		sdata := []byte(msg.Subject)

		copy(packet, []byte("$MAGIC8$"))
		binary.LittleEndian.PutUint32(packet[8:], counter)
		binary.LittleEndian.PutUint32(packet[12:], uint32(len(sdata)))
		binary.LittleEndian.PutUint32(packet[16:], uint32(len(msg.Data)))
		copy(packet[20:], sdata)
		udpconn.Write(packet[:len(sdata)+20])
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
			udpconn.Write(packet[:packetsize+8])
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
