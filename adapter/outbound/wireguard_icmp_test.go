package outbound

import (
	"context"
	"encoding/binary"
	"errors"
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/metacubex/mipstack"
)

// A peer's ICMP error must not end a UDP socket's reads: one socket talks to
// many peers (uTP, DHT), and the tunnel ends the whole session on a read
// error. Without tolerateICMP the error comes back from the read -- the
// mechanism -- and with it the next datagram does.
func TestWireGuardUDPSurvivesICMP(t *testing.T) {
	local := netip.MustParseAddr("10.8.1.3")
	unreachable := netip.MustParseAddrPort("192.0.2.1:9")
	peer := netip.MustParseAddrPort("198.51.100.7:4444")

	for _, tolerant := range []bool{false, true} {
		stack, err := mipstack.New(mipstack.Config{LocalAddresses: []netip.Prefix{netip.PrefixFrom(local, 32)}, MTU: 1420})
		if err != nil {
			t.Fatal(err)
		}
		if err := stack.Start(); err != nil {
			t.Fatal(err)
		}
		raw, err := stack.ListenUDP(context.Background(), "udp", netip.AddrPort{})
		if err != nil {
			t.Fatal(err)
		}
		pc := raw
		if tolerant {
			pc = tolerateICMP(raw)
		}
		port := uint16(raw.LocalAddr().(*net.UDPAddr).Port)

		// a datagram to a peer that is gone, as the stack sends it out
		if _, err := pc.WriteTo([]byte("hello"), net.UDPAddrFromAddrPort(unreachable)); err != nil {
			t.Fatal(err)
		}
		buf, sizes := [][]byte{make([]byte, 2048)}, []int{0}
		if _, err := stack.Read(buf, sizes, 0); err != nil {
			t.Fatal(err)
		}
		sent := buf[0][:sizes[0]]

		// the peer's host answers "port unreachable" while nothing else is
		// queued -- the tunnel reads as fast as datagrams come, and a queued
		// error is returned after any queued datagram
		icmp := append([]byte{3, 3, 0, 0, 0, 0, 0, 0}, sent...)
		binary.BigEndian.PutUint16(icmp[2:], checksum(icmp, 0))
		if _, err := stack.Write([][]byte{ipv4Packet(unreachable.Addr(), local, 1, icmp)}, 0); err != nil {
			t.Fatal(err)
		}
		b := make([]byte, 64)
		if !tolerant {
			_ = pc.SetReadDeadline(time.Now().Add(2 * time.Second))
			n, from, err := pc.ReadFrom(b)
			var icmpErr mipstack.ICMPError
			if !errors.As(err, &icmpErr) {
				t.Errorf("without tolerateICMP: %q from %v, %v; want the ICMP error", b[:n], from, err)
			}
		} else {
			// then another peer sends data: it is what the read returns
			data := udpDatagram(peer, netip.AddrPortFrom(local, port), []byte("world"))
			go func() {
				time.Sleep(50 * time.Millisecond)
				_, _ = stack.Write([][]byte{ipv4Packet(peer.Addr(), local, 17, data)}, 0)
			}()
			_ = pc.SetReadDeadline(time.Now().Add(2 * time.Second))
			n, from, err := pc.ReadFrom(b)
			if err != nil || string(b[:n]) != "world" || from.String() != peer.String() {
				t.Errorf("with tolerateICMP: %q from %v, %v; want the datagram", b[:n], from, err)
			}

			// the read the tunnel uses: the buffered one, kept by the wrapper
			wb, ok := pc.(interface {
				ReadFromWithBuffer(func(int) []byte) (int, net.Addr, error)
			})
			if !ok {
				t.Fatal("the wrapper lost ReadFromWithBuffer: the tunnel would fall back to a copying read")
			}
			if _, err := pc.WriteTo([]byte("again"), net.UDPAddrFromAddrPort(unreachable)); err != nil {
				t.Fatal(err)
			}
			if _, err := stack.Read(buf, sizes, 0); err != nil {
				t.Fatal(err)
			}
			icmp = append([]byte{3, 3, 0, 0, 0, 0, 0, 0}, buf[0][:sizes[0]]...)
			binary.BigEndian.PutUint16(icmp[2:], checksum(icmp, 0))
			if _, err := stack.Write([][]byte{ipv4Packet(unreachable.Addr(), local, 1, icmp)}, 0); err != nil {
				t.Fatal(err)
			}
			go func() {
				time.Sleep(50 * time.Millisecond)
				_, _ = stack.Write([][]byte{ipv4Packet(peer.Addr(), local, 17, data)}, 0)
			}()
			var got []byte
			n, from, err = wb.ReadFromWithBuffer(func(size int) []byte { got = make([]byte, size); return got })
			if err != nil || string(got[:n]) != "world" || from.String() != peer.String() {
				t.Errorf("buffered read with tolerateICMP: %q from %v, %v; want the datagram", got[:n], from, err)
			}
		}
		_ = pc.Close()
		_ = stack.Close()
	}
}

func ipv4Packet(src, dst netip.Addr, proto byte, payload []byte) []byte {
	h := make([]byte, 20, 20+len(payload))
	h[0] = 0x45
	binary.BigEndian.PutUint16(h[2:], uint16(20+len(payload)))
	h[8], h[9] = 64, proto
	s, d := src.As4(), dst.As4()
	copy(h[12:], s[:])
	copy(h[16:], d[:])
	binary.BigEndian.PutUint16(h[10:], checksum(h, 0))
	return append(h, payload...)
}

func udpDatagram(src, dst netip.AddrPort, data []byte) []byte {
	u := make([]byte, 8, 8+len(data))
	binary.BigEndian.PutUint16(u[0:], src.Port())
	binary.BigEndian.PutUint16(u[2:], dst.Port())
	binary.BigEndian.PutUint16(u[4:], uint16(8+len(data)))
	u = append(u, data...)
	s, d := src.Addr().As4(), dst.Addr().As4()
	pseudo := append(append(append([]byte{}, s[:]...), d[:]...), 0, 17, byte(len(u)>>8), byte(len(u)))
	binary.BigEndian.PutUint16(u[6:], checksum(u, sum(pseudo)))
	return u
}

func sum(b []byte) uint32 {
	var s uint32
	for i := 0; i+1 < len(b); i += 2 {
		s += uint32(binary.BigEndian.Uint16(b[i:]))
	}
	if len(b)%2 == 1 {
		s += uint32(b[len(b)-1]) << 8
	}
	return s
}

func checksum(b []byte, initial uint32) uint16 {
	s := initial + sum(b)
	for s>>16 != 0 {
		s = s&0xffff + s>>16
	}
	return ^uint16(s)
}
