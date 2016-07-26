// Copyright (c) 2014-2016 Bitmark Inc.
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
//
// prefix is optional and can be empty ("")
// returns prefixed string and IPv6 flag
func CanonicalIPandPort(prefix string, hostPort string) (string, bool, error) {

	host, port, err := net.SplitHostPort(hostPort)

	IP := net.ParseIP(strings.Trim(host, " "))
	if nil == IP {
		return "", false, fault.ErrInvalidIPAddress
	}

	numericPort, err := strconv.Atoi(strings.Trim(port, " "))
	if nil != err {
		return "", false, err
	}
	if numericPort < 1 || numericPort > 65535 {
		return "", false, fault.ErrInvalidPortNumber
	}

	if nil != IP.To4() {
		return prefix + IP.String() + ":" + strconv.Itoa(numericPort), false, nil
	}
	return prefix + "[" + IP.String() + "]:" + strconv.Itoa(numericPort), true, nil
}
