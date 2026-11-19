package main

import (
	"dd-nats/common/ddsvc"
	"dd-nats/common/udpproto"
	"errors"
	"log"
	"net"
	"time"
)

var udpconn *net.UDPConn

func main() {
	if usvc := ddsvc.InitService("dd-nats-outer-proxy"); usvc != nil {
		usvc.RunService(runEngine)
	}

	log.Printf("Exiting ...")
}

func runEngine(svc *ddsvc.DdUsvc) {
	port := svc.GetInt("port", 4359)
	bindip := svc.Get("bindip", "0.0.0.0")
	if err := listenUDP(svc, bindip, port); err != nil {
		svc.Error("Failed to listen", "Exiting application due to UDP connection failure, err: %s", err.Error())
		return
	}

	// A larger receive buffer reduces packet loss during bursts
	if err := udpconn.SetReadBuffer(svc.GetInt("readbuffer", 8*1024*1024)); err != nil {
		svc.Error("Outer proxy", "Failed to set UDP read buffer size: %s", err.Error())
	}

	svc.OnStop(func() { udpconn.Close() })

	prefix := svc.Get("prefix", "inner.")
	maxsize := svc.GetInt("maxmsgsize", udpproto.DefaultMaxMsgSize)
	reassembler := udpproto.NewReassembler(uint32(maxsize), 30*time.Second)

	// Start receiving UDP messages
	go readUDP(svc, prefix, reassembler)
	go purgeActiveMsgs(svc, reassembler)
}

func listenUDP(svc *ddsvc.DdUsvc, bindip string, port int) (err error) {
	if bindip == "nil" {
		svc.Trace("Outer proxy", "Listening to address (%s): nil:%d", bindip, port)
		udpconn, err = net.ListenUDP("udp4", nil)
		return err
	}

	addr := net.UDPAddr{
		Port: port,
		IP:   net.ParseIP(bindip),
	}

	svc.Trace("Outer proxy", "Listening to address: '%s:%d'", bindip, port)
	udpconn, err = net.ListenUDP("udp4", &addr)
	return err
}

func readUDP(svc *ddsvc.DdUsvc, prefix string, reassembler *udpproto.Reassembler) {
	// Big enough for any UDP datagram, the actual packet size is given by the read length
	buffer := make([]byte, 65536)

	for {
		n, from, err := udpconn.ReadFromUDP(buffer)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				svc.Info("Outer proxy", "UDP listener closed: %s", err.Error())
				return
			}

			svc.Error("Failed to read UDP", "Failed to read data packet, err: %s", err.Error())
			continue
		}

		msg, err := reassembler.Add(buffer[:n])
		if err != nil {
			svc.Error("Outer proxy", "Discarding packet from %s: %s", from.IP.String(), err.Error())
			continue
		}

		if msg != nil {
			if err := svc.Publish(prefix+msg.Subject, msg.Payload); err != nil {
				svc.Error("Outer proxy", "Failed to publish message, subject: %s, error: %s", msg.Subject, err.Error())
			}
		}
	}
}

// Remove partial messages that will never complete (lost packets)
func purgeActiveMsgs(svc *ddsvc.DdUsvc, reassembler *udpproto.Reassembler) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if count := reassembler.Purge(); count > 0 {
			svc.Error("Outer proxy", "Discarded %d incomplete message(s) (lost packets)", count)
		}

		if active := reassembler.Active(); active > 0 {
			svc.Trace("Outer proxy", "There are %d active messages", active)
		}
	}
}
