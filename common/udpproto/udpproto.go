// Package udpproto implements the packet format used to forward messages
// over the data diode (UDP, one way) between the inner and outer proxies.
//
// packet layout (all integers little endian)
// | uint32 | uint32 | uint32  | uint32 | uint32  | [subjectsize]byte | uint32  | [payloadsize]byte   |
// |--------|--------|---------|--------|---------|-------------------|---------|---------------------|
// | msgid  | total  | total   | packet | subject | subject...        | payload | payload fragment... |
// |        | size   | packets | no     | size    | (packetno = 0)    | size    |                     |
//
// Only the first packet (packetno = 0) carries the subject, subsequent packets
// have subject size 0.
package udpproto

import (
	"encoding/binary"
	"fmt"
	"sync"
	"time"
)

const (
	HeaderSize        = 5 * 4 // msgid, totalsize, totalpackets, packetno, subjectsize
	PayloadSizeField  = 4
	DefaultPacketSize = 1480
	DefaultMaxMsgSize = 64 * 1024 * 1024
)

// PacketCount returns the number of packets needed to send a payload of size
// 'size' with a subject of length 'ssize', given the max packet length
func PacketCount(size int, ssize int, packetlen int) (int, error) {
	firstcap := packetlen - HeaderSize - PayloadSizeField - ssize
	if firstcap <= 0 {
		return 0, fmt.Errorf("subject too long (%d bytes) for packet size %d", ssize, packetlen)
	}

	if size <= firstcap {
		return 1, nil
	}

	othercap := packetlen - HeaderSize - PayloadSizeField
	remaining := size - firstcap
	return 1 + (remaining+othercap-1)/othercap, nil
}

// Fragment splits a message into one or more packets, each at most packetlen
// bytes. The emit function is called for each packet in order, the packet
// buffer is reused between calls and must not be retained by emit.
func Fragment(msgid uint32, subject string, payload []byte, packetlen int, emit func(packet []byte) error) error {
	sdata := []byte(subject)
	totalsize := len(payload)

	totalpackets, err := PacketCount(totalsize, len(sdata), packetlen)
	if err != nil {
		return err
	}

	packet := make([]byte, packetlen)
	index := 0

	for packetno := 0; packetno < totalpackets; packetno++ {
		ssize := 0
		if packetno == 0 {
			ssize = len(sdata)
		}

		binary.LittleEndian.PutUint32(packet[0:], msgid)
		binary.LittleEndian.PutUint32(packet[4:], uint32(totalsize))
		binary.LittleEndian.PutUint32(packet[8:], uint32(totalpackets))
		binary.LittleEndian.PutUint32(packet[12:], uint32(packetno))
		binary.LittleEndian.PutUint32(packet[16:], uint32(ssize))
		copy(packet[HeaderSize:], sdata[:ssize])

		headersize := HeaderSize + ssize
		payloadsize := totalsize - index
		if space := packetlen - headersize - PayloadSizeField; payloadsize > space {
			payloadsize = space
		}

		binary.LittleEndian.PutUint32(packet[headersize:], uint32(payloadsize))
		copy(packet[headersize+PayloadSizeField:], payload[index:index+payloadsize])
		index += payloadsize

		if err := emit(packet[:headersize+PayloadSizeField+payloadsize]); err != nil {
			return err
		}
	}

	return nil
}

// Message is a fully reassembled message
type Message struct {
	MsgId   uint32
	Subject string
	Payload []byte
}

type partialMessage struct {
	subject      string
	totalsize    uint32
	totalpackets uint32
	lastpacket   uint32
	index        uint32
	payload      []byte
	lastSeen     time.Time
}

// Reassembler collects packets and reassembles complete messages. It
// validates all sizes in the packets and never trusts the content.
type Reassembler struct {
	MaxMsgSize uint32
	Timeout    time.Duration

	lock   sync.Mutex
	active map[uint32]*partialMessage
}

func NewReassembler(maxMsgSize uint32, timeout time.Duration) *Reassembler {
	if maxMsgSize == 0 {
		maxMsgSize = DefaultMaxMsgSize
	}

	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	return &Reassembler{MaxMsgSize: maxMsgSize, Timeout: timeout, active: make(map[uint32]*partialMessage)}
}

// Add processes one received packet. It returns a message when the last
// packet of a message has been received, nil if more packets are needed, or
// an error if the packet is malformed or out of sequence.
func (r *Reassembler) Add(pkt []byte) (*Message, error) {
	n := uint32(len(pkt))
	if n < HeaderSize+PayloadSizeField {
		return nil, fmt.Errorf("packet too short: %d bytes", n)
	}

	msgid := binary.LittleEndian.Uint32(pkt[0:])
	totalsize := binary.LittleEndian.Uint32(pkt[4:])
	totalpackets := binary.LittleEndian.Uint32(pkt[8:])
	packetno := binary.LittleEndian.Uint32(pkt[12:])
	subjectsize := binary.LittleEndian.Uint32(pkt[16:])

	if totalpackets == 0 || packetno >= totalpackets {
		return nil, fmt.Errorf("invalid packet number %d of %d (msgid %d)", packetno, totalpackets, msgid)
	}

	if totalsize > r.MaxMsgSize {
		return nil, fmt.Errorf("message too large: %d bytes (max %d, msgid %d)", totalsize, r.MaxMsgSize, msgid)
	}

	// Check sizes without risking integer overflow
	if subjectsize > n-HeaderSize-PayloadSizeField {
		return nil, fmt.Errorf("invalid subject size %d in packet of %d bytes (msgid %d)", subjectsize, n, msgid)
	}

	headersize := HeaderSize + subjectsize
	chunksize := binary.LittleEndian.Uint32(pkt[headersize:])
	offset := headersize + PayloadSizeField
	if chunksize > n-offset {
		return nil, fmt.Errorf("invalid payload size %d in packet of %d bytes (msgid %d)", chunksize, n, msgid)
	}

	chunk := pkt[offset : offset+chunksize]

	r.lock.Lock()
	defer r.lock.Unlock()

	var msg *partialMessage
	if packetno == 0 {
		if subjectsize == 0 {
			return nil, fmt.Errorf("first packet without subject (msgid %d)", msgid)
		}

		msg = &partialMessage{
			subject:      string(pkt[HeaderSize:headersize]),
			totalsize:    totalsize,
			totalpackets: totalpackets,
			payload:      make([]byte, totalsize),
		}

		if totalpackets > 1 {
			r.active[msgid] = msg
		}
	} else {
		msg = r.active[msgid]
		if msg == nil {
			// The start of the message was lost (or timed out), nothing to do
			return nil, nil
		}

		if packetno != msg.lastpacket+1 || totalpackets != msg.totalpackets || totalsize != msg.totalsize {
			delete(r.active, msgid)
			return nil, fmt.Errorf("packets out of sync: msgid %d, packet # %d, last packet %d, total size %d", msgid, packetno, msg.lastpacket, totalsize)
		}

		msg.lastpacket = packetno
	}

	msg.lastSeen = time.Now()

	if msg.index+chunksize > msg.totalsize {
		delete(r.active, msgid)
		return nil, fmt.Errorf("payload exceeds total size: msgid %d, index %d, chunk %d, total %d", msgid, msg.index, chunksize, msg.totalsize)
	}

	copy(msg.payload[msg.index:], chunk)
	msg.index += chunksize

	if packetno < totalpackets-1 {
		return nil, nil
	}

	delete(r.active, msgid)
	if msg.index != msg.totalsize {
		return nil, fmt.Errorf("incomplete message: msgid %d, received %d of %d bytes", msgid, msg.index, msg.totalsize)
	}

	return &Message{MsgId: msgid, Subject: msg.subject, Payload: msg.payload}, nil
}

// Purge removes partial messages that haven't received any packets within the
// timeout and returns the number of removed messages
func (r *Reassembler) Purge() int {
	r.lock.Lock()
	defer r.lock.Unlock()

	count := 0
	for id, msg := range r.active {
		if time.Since(msg.lastSeen) > r.Timeout {
			delete(r.active, id)
			count++
		}
	}

	return count
}

// Active returns the number of partially received messages
func (r *Reassembler) Active() int {
	r.lock.Lock()
	defer r.lock.Unlock()
	return len(r.active)
}
