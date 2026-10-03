package outbound

import (
	"encoding/binary"
	"net"
	"sync"
	"time"
)

// DPI Switch: a direct outbound that cuts the client's ClientHello so a DPI
// box reading the server name (SNI) off it cannot find the name.
//
// Measured on two Russian networks (03.10.2026): the box reassembles TCP --
// the hello cut into two segments is blocked -- and it reads several TLS
// records within one segment -- the hello cut into two records sent as one
// segment is blocked too, as are two records each in a segment of its own.
// What gets through everywhere: the hello cut into two records in the middle
// of the name, and the TCP segment boundary inside the FIRST record. The
// server sees a ClientHello in two records, which TLS allows (RFC 8446 5.1)
// and every server tried accepted.

// splitWait: how long a hello begun but not finished is held. A client sends
// its whole hello at once; this only keeps a stray protocol that happens to
// start like TLS from hanging.
var splitWait = time.Second

// maxRecord: the largest TLS plaintext record (RFC 8446 5.1)
const maxRecord = 1 << 14

type splitConn struct {
	net.Conn
	mu    sync.Mutex
	done  bool   // the first message has gone, as it was or cut
	buf   []byte // the first record, while it is incomplete
	timer *time.Timer
	err   error // the held bytes' write failed after splitWait
}

func newSplitConn(c net.Conn) net.Conn { return &splitConn{Conn: c} }

func (c *splitConn) Write(b []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return 0, c.err
	}
	if c.done {
		return c.Conn.Write(b)
	}
	c.buf = append(c.buf, b...)
	pieces, wait := splitHello(c.buf)
	if wait {
		if c.timer == nil {
			c.timer = time.AfterFunc(splitWait, c.flush)
		}
		return len(b), nil
	}
	c.done = true
	if c.timer != nil {
		c.timer.Stop()
	}
	c.buf = nil
	for _, p := range pieces {
		if _, err := c.Conn.Write(p); err != nil {
			return 0, err
		}
	}
	return len(b), nil
}

// flush sends what was held as it is: the hello never completed
func (c *splitConn) flush() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.done {
		return
	}
	c.done = true
	if _, err := c.Conn.Write(c.buf); err != nil {
		c.err = err
	}
	c.buf = nil
}

func (c *splitConn) Close() error {
	c.mu.Lock()
	if c.timer != nil {
		c.timer.Stop()
	}
	c.mu.Unlock()
	return c.Conn.Close()
}

// splitHello: the writes the bytes a client sent first go out as. wait: the
// first TLS record is not complete yet. Anything that is not a ClientHello
// carrying a server name in one record goes out as it came.
func splitHello(b []byte) (pieces [][]byte, wait bool) {
	asIs := [][]byte{b}
	// a TLS handshake record: 0x16, version 3.x, length
	if len(b) < 1 || b[0] != 0x16 {
		return asIs, false
	}
	if len(b) < 5 {
		return nil, true
	}
	n := int(binary.BigEndian.Uint16(b[3:5]))
	if b[1] != 3 || n == 0 || n > maxRecord {
		return asIs, false
	}
	if len(b) < 5+n {
		return nil, true
	}
	payload := b[5 : 5+n]
	at, size, ok := serverName(payload)
	if !ok {
		return asIs, false
	}
	cut := at + size/2
	r1 := make([]byte, 0, len(b)+5)
	r1 = append(r1, 0x16, b[1], b[2], byte(cut>>8), byte(cut))
	r1 = append(r1, payload[:cut]...)
	rest := n - cut
	r1 = append(r1, 0x16, b[1], b[2], byte(rest>>8), byte(rest))
	r1 = append(r1, payload[cut:]...)
	// whatever came after the record
	r1 = append(r1, b[5+n:]...)
	// the segment boundary inside the first record, before the cut
	seg := 5 + cut/2
	return [][]byte{r1[:seg], r1[seg:]}, false
}

// serverName: where the host name of the server_name extension lies in a
// handshake payload, if it is a ClientHello whole in it.
func serverName(p []byte) (at, size int, ok bool) {
	// handshake header: type 1 (client_hello), 24-bit length
	if len(p) < 4 || p[0] != 1 {
		return 0, 0, false
	}
	if int(p[1])<<16|int(p[2])<<8|int(p[3]) != len(p)-4 {
		// cut across records already, or followed by more: left alone
		return 0, 0, false
	}
	i := 4 + 2 + 32 // version, random
	skip := func(lenBytes int) bool {
		if i+lenBytes > len(p) {
			return false
		}
		l := 0
		for k := 0; k < lenBytes; k++ {
			l = l<<8 | int(p[i+k])
		}
		i += lenBytes + l
		return i <= len(p)
	}
	if !skip(1) || !skip(2) || !skip(1) { // session id, cipher suites, compression
		return 0, 0, false
	}
	if i+2 > len(p) {
		return 0, 0, false
	}
	end := i + 2 + int(binary.BigEndian.Uint16(p[i:]))
	if end > len(p) {
		return 0, 0, false
	}
	i += 2
	for i+4 <= end {
		typ := binary.BigEndian.Uint16(p[i:])
		l := int(binary.BigEndian.Uint16(p[i+2:]))
		data := i + 4
		if data+l > end {
			return 0, 0, false
		}
		if typ == 0 {
			// server_name_list: length, then entries of type, length, name
			if l < 5 || p[data+2] != 0 {
				return 0, 0, false
			}
			nl := int(binary.BigEndian.Uint16(p[data+3:]))
			if nl == 0 || data+5+nl > data+l {
				return 0, 0, false
			}
			return data + 5, nl, true
		}
		i = data + l
	}
	return 0, 0, false
}
