package main

import (
	"dd-nats/common/ddnats"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/nats-io/nats.go"
)

var udpconn *net.UDPConn

func main() {
	if err := listenUDP(); err != nil {
		log.Printf("Exiting application due to UDP connection failure, err: %s", err.Error())
		return
	}

	nc, err := ddnats.Connect(nats.DefaultURL)
	if err != nil {
		log.Printf("Exiting application due to NATS connection failure, err: %s", err.Error())
		return
	}

	go readUDP(nc)

	// Sleep until interrupted
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Printf("Exiting ...")
}

func listenUDP() (err error) {
	addr := net.UDPAddr{
		Port: 4359,
		IP:   net.ParseIP("0.0.0.0"),
	}

	udpconn, err = net.ListenUDP("udp", &addr)
	if err != nil {
		log.Printf("Failed to listen for UDP packets, err: %s\n", err.Error())
		return
	}

	return nil
}

func readUDP(nc *nats.Conn) {
	packetsize := 1200
	packet := make([]byte, packetsize)

	// Look for $MAGIC8$
	for {
		_, _, err := udpconn.ReadFromUDP(packet)
		if err != nil {
			// log.Printf("Failed to read MAGIC8 packet, err: %s", err.Error())
			continue
		}

		prevcounter, _, dlen, subject, err := parseMagic8Packet(packet)
		// prevcounter, _, dlen, _, err := parseMagic8Packet(packet)
		if err != nil {
			log.Printf("Error while parsing MAGIC8 packet, err: %s", err.Error())
			continue
		}

		// log.Printf("Found Magic8 with subject: %s", subject)
		index := uint32(0)
		mdata := make([]byte, dlen)

		for index < dlen {
			n, _, err := udpconn.ReadFromUDP(packet)
			if err != nil {
				log.Printf("Failed to read data packet, err: %s", err.Error())
				break
			}

			if n <= 0 {
				log.Printf("Failed to read data packet, n <= 0: %d", n)
				break
			}

			counter := binary.LittleEndian.Uint32(packet)
			size := binary.LittleEndian.Uint32(packet[4:])
			if int(size) > packetsize-8 {
				log.Printf("Invalid content size: %d, packet as string: %s", size, string(packet))
				break
			}

			if counter < prevcounter {
				log.Printf("Packets received in the wrong order, prev: %d, this: %d", prevcounter, counter)
			}

			if counter-prevcounter > 1 {
				log.Printf("Missing packets, prev: %d, this: %d, diff: %d", prevcounter, counter, counter-prevcounter)
			}

			prevcounter = counter

			copy(mdata[index:], packet[8:size+8])
			index += uint32(size)
		}

		nc.Publish(subject, mdata)
		// log.Printf("published subject: %s, data: %s", subject, string(mdata))
		// log.Printf("published subject: %s", subject)
	}

}

func parseMagic8Packet(packet []byte) (uint32, uint32, uint32, string, error) {
	if len(packet) < 10 {
		return 0, 0, 0, "system.error", fmt.Errorf("MAGIC8 packet size less than 10 bytes, %d", len(packet))
	}

	magic8 := string(packet[:8])
	if magic8 != "$MAGIC8$" {
		return 0, 0, 0, "system.error", fmt.Errorf("Packet is not a MAGIC8 packet, %s", magic8)
	}

	counter := binary.LittleEndian.Uint32(packet[8:])
	slen := binary.LittleEndian.Uint32(packet[12:])
	dlen := binary.LittleEndian.Uint32(packet[16:])
	subject := string(packet[20 : 20+slen])

	return counter, slen, dlen, subject, nil
}

func readUDP2(nc *nats.Conn) {

	// Look for $MAGIC8$
	magic8 := make([]byte, 8)

	for {
		n, _, err := udpconn.ReadFromUDP(magic8)
		if n != 8 {
			log.Printf("Failed to read MAGIC8, n: %d", n)
			continue
		}

		if err != nil {
			log.Printf("Failed to read MAGIC8, err: %s", err.Error())
			continue
		}

		len, err := readHeader()
		if err != nil {
			continue
		}

		sdata := make([]byte, len)
		n, _, err = udpconn.ReadFromUDP(sdata)
		if err != nil {
			log.Printf("Failed to read subject, err: %s", err.Error())
			continue
		}

		subject := string(sdata)

		len, err = readHeader()
		if err != nil {
			continue
		}

		mdata := make([]byte, len)
		n, _, err = udpconn.ReadFromUDP(mdata)
		if err != nil {
			log.Printf("Failed to read message body, err: %s", err.Error())
			continue
		}

		go nc.Publish(subject, mdata)
		// go log.Printf("published subject: %s, data: %s", subject, string(mdata))
	}

}

func readHeader() (uint32, error) {
	header := make([]byte, 4)
	n, _, err := udpconn.ReadFromUDP(header)
	if n != 4 {
		log.Printf("Failed to read header, n: %d", n)
		return 0, err
	}

	if err != nil {
		log.Printf("Failed to read header, err: %s", err.Error())
		return 0, err
	}

	slen := binary.LittleEndian.Uint32(header)
	return slen, nil
}
