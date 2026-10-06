package outbound

import (
	"bytes"
	"net"
	"testing"
)

type recordConn struct {
	net.PacketConn
	sent [][]byte
}

func (c *recordConn) WriteTo(b []byte, _ net.Addr) (int, error) {
	c.sent = append(c.sent, append([]byte(nil), b...))
	return len(b), nil
}

func TestQuicFakeBlob(t *testing.T) {
	if !isQuicInitial(quicFake) || len(quicFake) != 1280 {
		t.Fatalf("the decoy is no QUIC v1 Initial of 1280 bytes: %d bytes, % .8x", len(quicFake), quicFake)
	}
}

// The decoy goes once per destination, ahead of its first Initial on 443,
// and nowhere else
func TestQuicFakeConn(t *testing.T) {
	initial := append([]byte{0xc3, 0, 0, 0, 1}, make([]byte, 1200)...)
	short := append([]byte{0x40}, make([]byte, 100)...)
	v2 := append([]byte{0xd3, 0x6b, 0x33, 0x43, 0xcf}, make([]byte, 1200)...)
	a := &net.UDPAddr{IP: net.IPv4(142, 251, 1, 1), Port: 443}
	b := &net.UDPAddr{IP: net.IPv4(142, 251, 1, 2), Port: 443}
	other := &net.UDPAddr{IP: net.IPv4(142, 251, 1, 1), Port: 8443}
	rc := &recordConn{}
	c := newQuicFakeConn(rc)
	for _, w := range []struct {
		b    []byte
		addr net.Addr
	}{
		{initial, a}, {initial, a}, {short, a}, {initial, b}, {initial, other}, {v2, b}, {short, b},
	} {
		if n, err := c.WriteTo(w.b, w.addr); err != nil || n != len(w.b) {
			t.Fatalf("write: %d, %v", n, err)
		}
	}
	want := [][]byte{quicFake, initial, initial, short, quicFake, initial, initial, v2, short}
	if len(rc.sent) != len(want) {
		t.Fatalf("%d datagrams sent, want %d", len(rc.sent), len(want))
	}
	for i := range want {
		if !bytes.Equal(rc.sent[i], want[i]) {
			t.Errorf("datagram %d: % x…", i, rc.sent[i][:5])
		}
	}
}
