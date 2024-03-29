// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package util

import (
	"net"
	"strconv"
	"strings"

	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/logger"
)

// Connection - type to hold an IP and Port
type Connection struct {
	ip   net.IP
	port uint16
}

// NewConnection - create a connection from an Host:Port string
func NewConnection(hostPort string) (*Connection, error) {
	host, port, err := net.SplitHostPort(hostPort)
	if err != nil {
		return nil, err
	}

	host = strings.TrimSpace(host)

	IP := net.ParseIP(host)
	if host == "*" && IP == nil {
		IP = net.ParseIP("::")
	}
	if IP == nil {
		ips, err := net.LookupIP(host)
		if err != nil {
			return nil, err
		}
		if len(ips) < 1 {
			return nil, fault.InvalidIpAddress
		}
		IP = ips[0]
	}

	numericPort, err := strconv.Atoi(strings.TrimSpace(port))
	if err != nil {
		return nil, err
	}
	if numericPort < 1 || numericPort > 65535 {
		return nil, fault.InvalidPortNumber
	}
	c := &Connection{
		ip:   IP,
		port: uint16(numericPort),
	}
	return c, nil
}

// NewConnections -  convert an array of connections
func NewConnections(hostPort []string) ([]*Connection, error) {
	if len(hostPort) == 0 {
		return nil, fault.InvalidLength
	}
	c := make([]*Connection, len(hostPort))
	for i, hp := range hostPort {
		var err error
		c[i], err = NewConnection(hp)
		if err != nil {
			return nil, err
		}
	}
	return c, nil
}

// ConnectionFromIPandPort - convert an IP and port to a connection
func ConnectionFromIPandPort(ip net.IP, port uint16) *Connection {
	return &Connection{
		ip:   ip,
		port: port,
	}
}

// CanonicalIPandPort - make the IP:Port into canonical string
//
// examples:
//
//	IPv4:  127.0.0.1:1234
//	IPv6:  [::1]:1234
//
// prefix is optional and can be empty ("")
// returns prefixed string and IPv6 flag
func (conn *Connection) CanonicalIPandPort(prefix string) (string, bool) {

	port := int(conn.port)
	if conn.ip.To4() != nil {
		return prefix + conn.ip.String() + ":" + strconv.Itoa(port), false
	}
	return prefix + "[" + conn.ip.String() + "]:" + strconv.Itoa(port), true
}

// basic string conversion
func (conn Connection) String() string {
	s, _ := conn.CanonicalIPandPort("")
	return s
}

// MarshalText - convert to text for JSON
func (conn Connection) MarshalText() ([]byte, error) {
	s, _ := conn.CanonicalIPandPort("")
	return []byte(s), nil
}

// PackedConnection - type for packed byte buffer IP and Port
type PackedConnection []byte

// Pack - pack an IP and Port into a byte buffer
func (conn *Connection) Pack() PackedConnection {
	b := []byte(conn.ip)
	length := len(b)
	if length != 4 && length != 16 {
		logger.Panicf("connection.Pack: invalid IP length: %d", length)
	}
	size := length + 3 // count++port.high++port.low++ip
	b2 := make([]byte, size)
	b2[0] = byte(size)           // 7 or 19
	b2[1] = byte(conn.port >> 8) // port high byte
	b2[2] = byte(conn.port)      // port low byte
	copy(b2[3:], b)              // 4 byte IPv4 or 16 byte IPv6
	return b2
}

// Unpack - unpack a byte buffer into an IP and Port
// returns nil if unpack fails
// if successful returns connection and number of bytes used
// so an array can be unpacked more easily
func (packed PackedConnection) Unpack() (*Connection, int) {
	if packed == nil {
		return nil, 0
	}
	count := len(packed)
	if count < 7 {
		return nil, 0
	}
	n := packed[0]
	if n != 7 && n != 19 { // only valid values
		return nil, 0
	}

	ip := make([]byte, n-3) // 4 or 16 bytes
	copy(ip, packed[3:n])
	c := &Connection{
		ip:   ip,
		port: uint16(packed[1])<<8 + uint16(packed[2]),
	}
	return c, int(n)
}

// Unpack46 - unpack first IPv4 and first IPv6 plus Port
func (packed PackedConnection) Unpack46() (*Connection, *Connection) {

	// only expect two
	ipv4Connection := (*Connection)(nil)
	ipv6Connection := (*Connection)(nil)

	for {
		conn, n := packed.Unpack()
		packed = packed[n:]

		if conn == nil {
			return ipv4Connection, ipv6Connection
		}

		if conn.ip.To4() != nil {
			if ipv4Connection == nil {
				ipv4Connection = conn
			}
		} else if ipv6Connection == nil {
			ipv6Connection = conn
		}

		// if both kinds found
		if ipv4Connection != nil && ipv6Connection != nil {
			return ipv4Connection, ipv6Connection
		}
	}
}
