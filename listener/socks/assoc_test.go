package socks

import (
	"io"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/metacubex/mihomo/adapter/inbound"
	"github.com/metacubex/mihomo/component/auth"
	C "github.com/metacubex/mihomo/constant"
	authStore "github.com/metacubex/mihomo/listener/auth"
	LC "github.com/metacubex/mihomo/listener/config"
	"github.com/metacubex/mihomo/transport/socks5"
)

// udpCounter: a tunnel that counts the UDP packets handed to it
type udpCounter struct {
	mu sync.Mutex
	n  int
}

func (t *udpCounter) HandleTCPConn(conn net.Conn, _ *C.Metadata) { conn.Close() }
func (t *udpCounter) HandleUDPPacket(p C.UDPPacket, _ *C.Metadata) {
	t.mu.Lock()
	t.n++
	t.mu.Unlock()
	p.Drop()
}
func (t *udpCounter) NatTable() C.NatTable { return nil }
func (t *udpCounter) count() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.n
}

// DPI Switch: a listener with users takes UDP only from the source an
// authenticated association named, while its connection lasts; one that
// names no port is refused. See assoc.go.
func TestUDPAssociation(t *testing.T) {
	store := authStore.NewAuthStore(auth.NewAuthenticator([]auth.AuthUser{{User: "u", Pass: "p"}}))
	config := LC.AuthServer{Enable: true, Listen: "127.0.0.1:0", AuthStore: store}
	tun := &udpCounter{}
	add := []inbound.Addition{inbound.WithInName("probe")}
	tl, err := NewWithConfig(config, inbound.NewListenConfig(), tun, add...)
	if err != nil {
		t.Fatal(err)
	}
	defer tl.Close()
	ul, err := NewUDPWithConfig(config, inbound.NewListenConfig(), tun, add...)
	if err != nil {
		t.Fatal(err)
	}
	defer ul.Close()
	relay, _ := net.ResolveUDPAddr("udp", ul.Address())

	client, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	packet, _ := socks5.EncodeUDPPacket(socks5.ParseAddr("192.0.2.1:53"), []byte("hello"))
	send := func() int {
		before := tun.count()
		if _, err := client.WriteTo(packet, relay); err != nil {
			t.Fatal(err)
		}
		time.Sleep(100 * time.Millisecond)
		return tun.count() - before
	}
	associate := func(port int) net.Conn {
		c, err := net.Dial("tcp", tl.Address())
		if err != nil {
			t.Fatal(err)
		}
		_, err = socks5.ClientHandshake(c, socks5.ParseAddr("127.0.0.1:"+strconv.Itoa(port)), socks5.CmdUDPAssociate, &socks5.User{Username: "u", Password: "p"})
		if err != nil {
			t.Fatal(err)
		}
		return c
	}

	if n := send(); n != 0 {
		t.Fatalf("taken with no association: %d", n)
	}
	c := associate(client.LocalAddr().(*net.UDPAddr).Port)
	if n := send(); n != 1 {
		t.Fatalf("not taken with the association named: %d", n)
	}
	c.Close()
	time.Sleep(100 * time.Millisecond)
	if n := send(); n != 0 {
		t.Fatalf("taken after the association ended: %d", n)
	}

	// no port named: the connection goes, nothing is taken
	c = associate(0)
	c.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, err := c.Read(make([]byte, 1)); err != io.EOF {
		t.Fatalf("an association with no port kept: %v", err)
	}
	c.Close()
	if n := send(); n != 0 {
		t.Fatalf("taken with an association that named no port: %d", n)
	}

	// another source with a live association for this one: not taken
	c = associate(client.LocalAddr().(*net.UDPAddr).Port)
	defer c.Close()
	other, _ := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	defer other.Close()
	before := tun.count()
	other.WriteTo(packet, relay)
	time.Sleep(100 * time.Millisecond)
	if n := tun.count() - before; n != 0 {
		t.Fatalf("another source taken: %d", n)
	}
}
