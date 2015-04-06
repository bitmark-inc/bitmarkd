// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package util

import (
	"github.com/bitmark-inc/bitmarkd/fault"
	"net"
	"strconv"
	"strings"
)

// make the IP:Port canonical
//
// examples:
//   IPv4:  127.0.0.1:1234
//   IPv6:  [::1]:1234
func CanonicalIPandPort(hostPort string) (string, error) {

	host, port, err := net.SplitHostPort(hostPort)

	IP := net.ParseIP(strings.Trim(host, " "))
	if nil == IP {
		return "", fault.ErrInvalidIPAddress
	}

	numericPort, err := strconv.Atoi(strings.Trim(port, " "))
	if nil != err {
		return "", err
	}
	if numericPort < 1 || numericPort > 65535 {
		return "", fault.ErrInvalidPortNumber
	}

	if nil != IP.To4() {
		return IP.String() + ":" + strconv.Itoa(numericPort), nil
	}
	return "[" + IP.String() + "]:" + strconv.Itoa(numericPort), nil
}
