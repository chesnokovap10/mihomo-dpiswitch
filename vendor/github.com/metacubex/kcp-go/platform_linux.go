// The MIT License (MIT)
//
// Copyright (c) 2015 xtaci
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

//go:build linux

package kcp

import (
	"net"
	"syscall"

	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

type (
	platform struct {
		batchConn batchConn
	}

	// udpConn is an interface implemented by net.UDPConn.
	// It can be used for interface assertions to check if a net.Conn is a UDP connection.
	udpConn interface {
		SyscallConn() (syscall.RawConn, error)
		ReadMsgUDP(b, oob []byte) (n, oobn, flags int, addr *net.UDPAddr, err error)
	}

	// batchConn defines the interface used in batch IO
	batchConn interface {
		WriteBatch(ms []ipv4.Message, flags int) (int, error)
		ReadBatch(ms []ipv4.Message, flags int) (int, error)
	}
)

// Contrary to what the naming suggests, the ipv{4,6}.Message is not dependent on the IP version.
// They're both just aliases for x/net/internal/socket.Message.
// This means we can use this struct to read from a socket that receives both IPv4 and IPv6 messages.
var _ ipv4.Message = ipv6.Message{}

// newBatchConn creates a batchConn based on the IP version of the provided net.PacketConn.
func newBatchConn(conn net.PacketConn) batchConn {
	// Allows callers to pass in a connection that already satisfies batchConn interface
	// to make use of the optimisation. Otherwise, ipv4.NewPacketConn would unwrap the file descriptor
	// via SyscallConn(), and read it that way, which might not be what the caller wants.
	if ibc, ok := conn.(batchConn); ok {
		return ibc
	} else if _, ok := conn.(net.Conn); ok {
		if sConn, ok := conn.(syscall.Conn); ok {
			if _, err := sConn.SyscallConn(); err == nil {
				return ipv4.NewPacketConn(conn)
			}
		}
	}
	return nil
}

func (sess *UDPSession) initPlatform() {
	sess.platform.batchConn = newBatchConn(sess.conn)
}
