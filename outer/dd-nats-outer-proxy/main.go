package main

import (
	"dd-nats/common/ddsvc"
	"encoding/binary"
	"log"
	"net"
	"strconv"
	"time"
)

type msgInfo struct {
	msgid        uint32
	totalsize    uint32
	totalpackets uint32
	packetno     uint32
	lastpacket   uint32
	index        uint32
	subject      string
	payload      []byte
}

var udpconn *net.UDPConn
var activemsgs map[uint32]*msgInfo

func main() {
	var usvc *ddsvc.DdUsvc
	if usvc = ddsvc.InitService("dd-nats-outer-proxy"); usvc != nil {
		usvc.RunService(runEngine)
	}

	usvc.Trace("Application status", "Exiting ...")
}

func runEngine(svc *ddsvc.DdUsvc) {
	port, _ := strconv.Atoi(svc.Get("port", "4359"))
	if err := listenUDP(svc, port); err != nil {
		svc.Error("Failed to connect", "Exiting application due to UDP connection failure, err: %s", err.Error())
		return
	}

	prefix := svc.Get("prefix", "inner.")

	// Start receiving UDP messages
	go readUDP(svc, prefix)
	go readActiveMsgs()
}

func listenUDP(svc *ddsvc.DdUsvc, port int) (err error) {
	addr := net.UDPAddr{
		Port: port,
		IP:   net.ParseIP("0.0.0.0"),
	}

	udpconn, err = net.ListenUDP("udp", &addr)
	return err
}

// packet layout
// | uint32 | uint32 | uint32 | uint32 | uint32 | [subjectsize]byte | uint32 | [packetsize]byte    |
// |--------|--------|--------|--------|--------|-------------------|--------|---------------------|
// | msgid  | total  | total  | packet | subject| subject...        | payload| payload fragment... |
// |        | size   | packets| no     | size   | (packetno = 0)    | size   |                     |

func readUDP(svc *ddsvc.DdUsvc, prefix string) {
	packetlen := 1480
	packet := make([]byte, packetlen)
	activemsgs = make(map[uint32]*msgInfo)

	for {
		_, _, err := udpconn.ReadFromUDP(packet)
		if err != nil {
			svc.Error("Failed to read UDP", "Failed to read data packet, err: %s", err.Error())
			continue
		}

		// read message id, packet no and total packets to determine if it is a new message or not
		// and if we need to store it or not (single packets doesn't have to be kept)
		msgid := binary.LittleEndian.Uint32(packet)
		totalsize := binary.LittleEndian.Uint32(packet[4:])
		totalpackets := binary.LittleEndian.Uint32(packet[8:])
		packetno := binary.LittleEndian.Uint32(packet[12:])
		subjectsize := binary.LittleEndian.Uint32(packet[16:])
		msg := &msgInfo{msgid: msgid, totalsize: totalsize, totalpackets: totalpackets, packetno: packetno, lastpacket: packetno}

		// we only get subject with the first packet
		if packetno == 0 || subjectsize > 0 {
			msg.subject = string(packet[20 : 20+subjectsize])
			msg.payload = make([]byte, msg.totalsize)
		}

		// if totalpackets > 1, then we need to find the message or create a new
		if totalpackets > 1 {
			if packetno == 0 {
				activemsgs[msgid] = msg
			} else {
				// not a new message, find it
				msg = activemsgs[msgid] // Is this introducing too much lag?
				if msg == nil {
					continue
				}

				msg.lastpacket = msg.packetno
				msg.packetno = packetno

				if msg.packetno != msg.lastpacket+1 {
					svc.Error("Packets out of sync", "Attempting to synchronize: msgid %d, packet # %d, lastpacket %d, total size %d", msgid, msg.packetno, msg.lastpacket, totalsize)
					delete(activemsgs, msg.msgid)
					continue
				}
			}
		} else {
			msg.payload = make([]byte, msg.totalsize)
		}

		chunksize := binary.LittleEndian.Uint32(packet[20+subjectsize:])
		offset := 24 + subjectsize
		copy(msg.payload[msg.index:], packet[offset:offset+chunksize])
		msg.index += chunksize

		if msg.packetno == msg.totalpackets-1 {
			svc.Publish(prefix+msg.subject, msg.payload[:totalsize])
			delete(activemsgs, msg.msgid)
		}
	}
}

func readActiveMsgs() {
	ticker := time.NewTicker(1000 * time.Millisecond)
	for {
		<-ticker.C
		count := len(activemsgs)
		if count > 0 {
			log.Printf("There are %d active messages", count)
		}
	}
}
