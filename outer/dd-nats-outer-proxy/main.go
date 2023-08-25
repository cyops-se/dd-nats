package main

import (
	"dd-nats/common/ddnats"
	"dd-nats/common/ddsvc"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"
)

var udpconn *net.UDPConn

func main() {
	if svc := ddsvc.InitService("dd-nats-outer-proxy"); svc != nil {
		svc.RunService(runEngine)
	}

	log.Printf("Exiting ...")
}

func runEngine(svc *ddsvc.DdUsvc) {
	port, _ := strconv.Atoi(svc.Get("port", "4359"))
	if err := listenUDP(svc, port); err != nil {
		log.Printf("Exiting application due to UDP connection failure, err: %s", err.Error())
		return
	}

	prefix := svc.Get("prefix", "inner.")

	// Start receiving UDP messages
	go readUDP(svc, prefix)
}

func listenUDP(svc *ddsvc.DdUsvc, port int) (err error) {
	addr := net.UDPAddr{
		Port: port,
		IP:   net.ParseIP("0.0.0.0"),
	}

	udpconn, err = net.ListenUDP("udp", &addr)
	if err != nil {
		log.Printf("Failed to listen for UDP packets, err: %s\n", err.Error())
		return
	}

	return nil
}

func readUDP(svc *ddsvc.DdUsvc, prefix string) {
	packetsize := 1200
	packet := make([]byte, packetsize)
	count := 0
	total := uint64(0)

	// Look for $MAGIC8$
	for {
		start := time.Now().UnixNano()
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

		ddnats.PublishData(prefix+subject, mdata)

		count++
		if svc.Context.Trace {
			total += uint64(time.Now().UnixNano()) - uint64(start)
			if count%100 == 0 {
				log.Printf("One message takes %f nanoseconds in average", float64(total)/float64(count))
			}
		}
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
