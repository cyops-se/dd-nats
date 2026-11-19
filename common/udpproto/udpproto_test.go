package udpproto

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"testing"
	"time"
)

func fragment(t *testing.T, msgid uint32, subject string, payload []byte, packetlen int) [][]byte {
	t.Helper()
	var packets [][]byte
	err := Fragment(msgid, subject, payload, packetlen, func(p []byte) error {
		if len(p) > packetlen {
			t.Fatalf("packet larger than %d: %d", packetlen, len(p))
		}
		packets = append(packets, append([]byte(nil), p...))
		return nil
	})
	if err != nil {
		t.Fatalf("fragment failed: %v", err)
	}
	return packets
}

func TestRoundTripSizes(t *testing.T) {
	subjects := []string{"a", "process.actual", string(bytes.Repeat([]byte("s"), 200))}
	sizes := []int{0, 1, 1435, 1436, 1437, 1456, 1480, 1481, 2880, 2892, 2893, 256 * 1024, 350*1024 + 7}

	for _, subject := range subjects {
		for _, size := range sizes {
			payload := make([]byte, size)
			rand.Read(payload)

			packets := fragment(t, 42, subject, payload, DefaultPacketSize)
			expected, _ := PacketCount(size, len(subject), DefaultPacketSize)
			if len(packets) != expected {
				t.Fatalf("subject %d, size %d: got %d packets, expected %d", len(subject), size, len(packets), expected)
			}

			r := NewReassembler(0, time.Second)
			var msg *Message
			for i, p := range packets {
				m, err := r.Add(p)
				if err != nil {
					t.Fatalf("subject %d, size %d, packet %d: %v", len(subject), size, i, err)
				}
				if m != nil {
					if i != len(packets)-1 {
						t.Fatalf("message completed early at packet %d", i)
					}
					msg = m
				}
			}

			if msg == nil {
				t.Fatalf("subject %d, size %d: message not completed", len(subject), size)
			}
			if msg.Subject != subject || !bytes.Equal(msg.Payload, payload) {
				t.Fatalf("subject %d, size %d: payload mismatch", len(subject), size)
			}
			if r.Active() != 0 {
				t.Fatalf("active messages left: %d", r.Active())
			}
		}
	}
}

func TestSubjectTooLong(t *testing.T) {
	err := Fragment(1, string(bytes.Repeat([]byte("x"), DefaultPacketSize)), []byte("x"), DefaultPacketSize, func([]byte) error { return nil })
	if err == nil {
		t.Fatal("expected error for too long subject")
	}
}

func TestLostPacketDetected(t *testing.T) {
	payload := make([]byte, 10000)
	packets := fragment(t, 7, "subject", payload, DefaultPacketSize)
	r := NewReassembler(0, time.Second)

	for i, p := range packets {
		if i == 2 {
			continue // lose a packet
		}
		m, err := r.Add(p)
		if m != nil {
			t.Fatal("message should not complete with a lost packet")
		}
		if i == 3 && err == nil {
			t.Fatal("expected out of sync error")
		}
	}
}

func TestPurge(t *testing.T) {
	packets := fragment(t, 9, "subject", make([]byte, 5000), DefaultPacketSize)
	r := NewReassembler(0, time.Millisecond)
	r.Add(packets[0])
	if r.Active() != 1 {
		t.Fatal("expected one active message")
	}
	time.Sleep(5 * time.Millisecond)
	if n := r.Purge(); n != 1 || r.Active() != 0 {
		t.Fatalf("purge removed %d, active %d", n, r.Active())
	}
}

func TestMalformedPackets(t *testing.T) {
	r := NewReassembler(1024, time.Second)

	header := func(totalsize, totalpackets, packetno, subjectsize uint32) []byte {
		p := make([]byte, 64)
		binary.LittleEndian.PutUint32(p[4:], totalsize)
		binary.LittleEndian.PutUint32(p[8:], totalpackets)
		binary.LittleEndian.PutUint32(p[12:], packetno)
		binary.LittleEndian.PutUint32(p[16:], subjectsize)
		return p
	}

	cases := map[string][]byte{
		"short":            make([]byte, 10),
		"zero packets":     header(1, 0, 0, 1),
		"packetno too big": header(1, 1, 1, 1),
		"too large":        header(1<<30, 1, 0, 1),
		"subject overflow": header(1, 1, 0, 0xFFFFFFFF),
		"subject too big":  header(1, 1, 0, 100),
	}

	big := header(10, 1, 0, 1)
	binary.LittleEndian.PutUint32(big[21:], 0xFFFFFFF0) // payload size
	cases["payload too big"] = big

	exceed := header(1, 1, 0, 1)
	binary.LittleEndian.PutUint32(exceed[21:], 10) // payload larger than total size
	cases["payload exceeds total"] = exceed

	for name, p := range cases {
		if m, err := r.Add(p); err == nil || m != nil {
			t.Errorf("%s: expected error, got msg %v, err %v", name, m, err)
		}
	}
}

func FuzzReassembler(f *testing.F) {
	f.Add(make([]byte, 30))
	f.Fuzz(func(t *testing.T, data []byte) {
		r := NewReassembler(1<<20, time.Second)
		r.Add(data) // must never panic
	})
}
