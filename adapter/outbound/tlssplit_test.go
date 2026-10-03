package outbound

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"io"
	"math/big"
	"net"
	"testing"
	"time"
)

// clientHello: the first record a Go TLS client writes for this name
func clientHello(t *testing.T, name string) []byte {
	t.Helper()
	a, b := net.Pipe()
	go func() {
		c := tls.Client(a, &tls.Config{ServerName: name, InsecureSkipVerify: true})
		_ = c.Handshake()
	}()
	hdr := make([]byte, 5)
	if _, err := io.ReadFull(b, hdr); err != nil {
		t.Fatal(err)
	}
	rec := make([]byte, binary.BigEndian.Uint16(hdr[3:5]))
	if _, err := io.ReadFull(b, rec); err != nil {
		t.Fatal(err)
	}
	a.Close()
	b.Close()
	return append(hdr, rec...)
}

// records: the TLS records in b, as their payloads
func records(t *testing.T, b []byte) [][]byte {
	t.Helper()
	var out [][]byte
	for len(b) > 0 {
		if len(b) < 5 || b[0] != 0x16 {
			t.Fatalf("not a handshake record: % x", b)
		}
		n := int(binary.BigEndian.Uint16(b[3:5]))
		out = append(out, b[5:5+n])
		b = b[5+n:]
	}
	return out
}

// The hello goes as two records cut in the middle of the name, in two
// writes whose boundary is inside the first record -- the one shape the DPI
// boxes measured let through.
func TestSplitHello(t *testing.T) {
	const name = "www.instagram.com"
	hello := clientHello(t, name)
	pieces, wait := splitHello(hello)
	if wait || len(pieces) != 2 {
		t.Fatalf("pieces %d, wait %v", len(pieces), wait)
	}
	all := append(append([]byte(nil), pieces[0]...), pieces[1]...)
	recs := records(t, all)
	if len(recs) != 2 {
		t.Fatalf("%d records", len(recs))
	}
	if !bytes.Equal(append(append([]byte(nil), recs[0]...), recs[1]...), hello[5:]) {
		t.Fatal("the records do not add up to the hello")
	}
	// neither record holds the whole name, and the cut is inside it
	for i, r := range recs {
		if bytes.Contains(r, []byte(name)) {
			t.Errorf("record %d holds the whole name", i)
		}
	}
	if !bytes.HasSuffix(recs[0], []byte(name[:len(name)/2])) || !bytes.HasPrefix(recs[1], []byte(name[len(name)/2:])) {
		t.Error("not cut in the middle of the name")
	}
	if len(pieces[0]) <= 5 || len(pieces[0]) >= 5+len(recs[0]) {
		t.Errorf("first write %d bytes: not inside the first record (5..%d)", len(pieces[0]), 5+len(recs[0]))
	}
	// the hello's version bytes stay
	if all[1] != hello[1] || all[2] != hello[2] {
		t.Error("record version changed")
	}
}

func TestSplitHelloLeavesOthers(t *testing.T) {
	for name, b := range map[string][]byte{
		"http":         []byte("GET / HTTP/1.1\r\nHost: a\r\n\r\n"),
		"alert record": {0x15, 3, 3, 0, 2, 2, 40},
		"too long":     {0x16, 3, 1, 0x40, 0x01},
		"not 3.x":      {0x16, 2, 0, 0, 1, 1},
	} {
		pieces, wait := splitHello(b)
		if wait || len(pieces) != 1 || !bytes.Equal(pieces[0], b) {
			t.Errorf("%s: %d pieces, wait %v", name, len(pieces), wait)
		}
	}
	// a ClientHello without a name goes as it is
	hello := clientHello(t, "")
	if pieces, _ := splitHello(hello); len(pieces) != 1 {
		t.Errorf("no name: %d pieces", len(pieces))
	}
	// a hello begun waits for the rest
	if _, wait := splitHello(clientHello(t, "a.example")[:100]); !wait {
		t.Error("a hello cut short did not wait")
	}
}

// captureConn: what the outbound wrote, write by write
type captureConn struct {
	net.Conn
	writes [][]byte
	got    chan struct{}
}

func (c *captureConn) Write(b []byte) (int, error) {
	c.writes = append(c.writes, append([]byte(nil), b...))
	select {
	case c.got <- struct{}{}:
	default:
	}
	return len(b), nil
}

// The client's hello coming in two writes -- the sniffer let part of it
// through first -- is still cut as a whole, and what follows passes as is.
func TestSplitConnJoinsHello(t *testing.T) {
	hello := clientHello(t, "rutracker.org")
	cc := &captureConn{got: make(chan struct{}, 1)}
	c := newSplitConn(cc)
	half := len(hello) / 2
	if n, err := c.Write(hello[:half]); n != half || err != nil {
		t.Fatal(n, err)
	}
	if len(cc.writes) != 0 {
		t.Fatal("wrote part of the hello")
	}
	c.Write(hello[half:])
	c.Write([]byte("after"))
	if len(cc.writes) != 3 || string(cc.writes[2]) != "after" {
		t.Fatalf("writes: %d", len(cc.writes))
	}
	if recs := records(t, append(cc.writes[0], cc.writes[1]...)); len(recs) != 2 {
		t.Fatalf("%d records", len(recs))
	}
}

// A hello begun and never finished goes as it is after splitWait.
func TestSplitConnFlushes(t *testing.T) {
	was := splitWait
	splitWait = 20 * time.Millisecond
	defer func() { splitWait = was }()
	cc := &captureConn{got: make(chan struct{}, 1)}
	c := newSplitConn(cc)
	part := clientHello(t, "a.example")[:50]
	c.Write(part)
	select {
	case <-cc.got:
	case <-time.After(2 * time.Second):
		t.Fatal("nothing flushed")
	}
	c.Write([]byte("x"))
	if len(cc.writes) != 2 || !bytes.Equal(cc.writes[0], part) {
		t.Fatalf("writes: %q", cc.writes)
	}
}

// A real TLS server takes the cut hello: the handshake completes and data
// flows both ways.
func TestSplitConnHandshake(t *testing.T) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(1), DNSNames: []string{"split.example"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert := tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	sni := make(chan string, 1)
	go func() {
		raw, err := ln.Accept()
		if err != nil {
			return
		}
		s := tls.Server(raw, &tls.Config{Certificates: []tls.Certificate{cert},
			GetConfigForClient: func(h *tls.ClientHelloInfo) (*tls.Config, error) { sni <- h.ServerName; return nil, nil }})
		defer s.Close()
		buf := make([]byte, 4)
		if _, err := io.ReadFull(s, buf); err == nil {
			s.Write(buf)
		}
	}()
	raw, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	leaf, _ := x509.ParseCertificate(der)
	pool.AddCert(leaf)
	c := tls.Client(newSplitConn(raw), &tls.Config{ServerName: "split.example", RootCAs: pool})
	defer c.Close()
	c.SetDeadline(time.Now().Add(5 * time.Second))
	if _, err := c.Write([]byte("ping")); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 4)
	if _, err := io.ReadFull(c, buf); err != nil || string(buf) != "ping" {
		t.Fatalf("%q %v", buf, err)
	}
	if got := <-sni; got != "split.example" {
		t.Errorf("server saw the name %q", got)
	}
}
