package socks

// DPI Switch: a SOCKS listener with users of its own takes UDP only from
// the address an authenticated UDP ASSOCIATE named, and only for as long as
// that association's TCP connection lasts. mihomo took a datagram from
// whoever sent one to the port: the users kept TCP for those who knew them,
// and UDP for nobody -- on DPI Switch's prober listeners any program, of any
// account, could send UDP past the tunnel through the direct one.
//
// An association names the port it sends from, and the address is the one
// its TCP connection came from: on loopback every program shares the
// address, so the port is what tells them apart. One that names no port is
// refused on such a listener -- its datagrams could not be told from
// anyone's. DPI Switch's prober names its port. The default listener, and a
// listener with no users, take UDP as mihomo did.

import (
	"net"
	"net/netip"
	"sync"

	authStore "github.com/metacubex/mihomo/listener/auth"
	LC "github.com/metacubex/mihomo/listener/config"
	"github.com/metacubex/mihomo/transport/socks5"
)

// associations: the sources one listener takes UDP from, with how many live
// associations name each
type associations struct {
	mu   sync.Mutex
	from map[netip.AddrPort]int
}

// assocByListen: the tables by listen address -- the TCP and the UDP
// listener of one address are made apart, and meet here
var assocByListen sync.Map

// associationsFor: the table of a listener with users of its own, nil for
// the others
func associationsFor(config LC.AuthServer) *associations {
	if config.AuthStore == nil || config.AuthStore == authStore.Default || config.AuthStore.Authenticator() == nil {
		return nil
	}
	a, _ := assocByListen.LoadOrStore(config.Listen, &associations{from: map[netip.AddrPort]int{}})
	return a.(*associations)
}

func (a *associations) add(ap netip.AddrPort) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.from[ap]++
}

func (a *associations) remove(ap netip.AddrPort) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.from[ap]--; a.from[ap] <= 0 {
		delete(a.from, ap)
	}
}

// has: whether a datagram from addr belongs to a live association
func (a *associations) has(addr net.Addr) bool {
	ap, ok := addrPort(addr)
	if !ok {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.from[ap] > 0
}

// assocSource: where an association's datagrams come from -- the address
// of its TCP connection, the port it named; false for none named
func assocSource(peer net.Addr, named socks5.Addr) (netip.AddrPort, bool) {
	ap, ok := addrPort(peer)
	if !ok || len(named) < 2 {
		return netip.AddrPort{}, false
	}
	port := uint16(named[len(named)-2])<<8 | uint16(named[len(named)-1])
	if port == 0 {
		return netip.AddrPort{}, false
	}
	return netip.AddrPortFrom(ap.Addr(), port), true
}

func addrPort(addr net.Addr) (netip.AddrPort, bool) {
	var ap netip.AddrPort
	switch a := addr.(type) {
	case *net.UDPAddr:
		ap = a.AddrPort()
	case *net.TCPAddr:
		ap = a.AddrPort()
	default:
		if addr == nil {
			return netip.AddrPort{}, false
		}
		var err error
		if ap, err = netip.ParseAddrPort(addr.String()); err != nil {
			return netip.AddrPort{}, false
		}
	}
	if !ap.IsValid() {
		return netip.AddrPort{}, false
	}
	return netip.AddrPortFrom(ap.Addr().Unmap(), ap.Port()), true
}
