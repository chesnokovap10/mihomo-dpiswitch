package outbound

import (
	_ "embed"
	"net"
	"net/netip"
	"sync"
)

// DPI Switch: a direct outbound that sends a decoy QUIC Initial before the
// client's first one, so a DPI box reading the server name off the Initial
// reads the decoy's.
//
// Measured at home (06.10.2026): the box decrypts the Initial -- its keys
// come from the connection ID in the clear -- and silently drops the flow
// when the name is a blocked one (youtube.com, googlevideo.com, rutracker.org
// and so on; google.com on the same address goes through). It judges the
// flow by the first Initial it can read: one with www.google.com sent ahead
// lets the real one through, on every site tried. Random bytes ahead do
// not: the box skips what it cannot decrypt. The decoy reaches the server
// too, which answers it under the decoy's connection ID, and the client
// drops that as a connection it does not know.

// quicFake: a QUIC v1 Initial for www.google.com, padded to 1280 bytes, the
// ClientHello whole in it
//
//go:embed quicfake.bin
var quicFake []byte

// maxFaked: how many destinations one socket remembers having sent the decoy
// to; past it the set starts over, and a destination may get a second decoy
// ahead of a retransmitted Initial, which does no harm
const maxFaked = 64

type quicFakeConn struct {
	net.PacketConn
	mu    sync.Mutex
	faked map[netip.AddrPort]struct{}
}

func newQuicFakeConn(pc net.PacketConn) net.PacketConn {
	return &quicFakeConn{PacketConn: pc, faked: map[netip.AddrPort]struct{}{}}
}

func (c *quicFakeConn) WriteTo(b []byte, addr net.Addr) (int, error) {
	if isQuicInitial(b) {
		if ua, ok := addr.(*net.UDPAddr); ok && ua.Port == 443 {
			ap := ua.AddrPort()
			c.mu.Lock()
			_, done := c.faked[ap]
			if !done {
				if len(c.faked) >= maxFaked {
					c.faked = map[netip.AddrPort]struct{}{}
				}
				c.faked[ap] = struct{}{}
			}
			c.mu.Unlock()
			if !done {
				// a decoy that fails to go is no reason to hold the real one
				_, _ = c.PacketConn.WriteTo(quicFake, addr)
			}
		}
	}
	return c.PacketConn.WriteTo(b, addr)
}

// isQuicInitial: a QUIC version 1 long-header Initial packet (RFC 9000 17.2.2)
func isQuicInitial(b []byte) bool {
	return len(b) >= 7 && b[0]&0xf0 == 0xc0 &&
		b[1] == 0 && b[2] == 0 && b[3] == 0 && b[4] == 1
}
