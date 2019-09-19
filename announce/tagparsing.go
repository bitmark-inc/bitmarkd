// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package announce

import (
	"encoding/hex"
	"net"
	"strconv"
	"strings"

	"github.com/bitmark-inc/bitmarkd/fault"
)

// the tag to detect applicable TXT records from DNS
var supportedTags = map[string]struct{}{
	"bitmark=v2": {},
	"bitmark=v3": {},
}

const (
	publicKeyLength   = 2 * 32 // characters
	fingerprintLength = 2 * 32 // characters
)

type tagline struct {
	ipv4                   net.IP
	ipv6                   net.IP
	rpcPort                uint16
	connectPort            uint16
	certificateFingerprint []byte
	peerID                 string
}

// decode DNS TXT records of these forms
//
//   <TAG> a=<IPv4;IPv6> c=<PORT> r=<PORT> f=<SHA3-256(cert)> p=<PUBLIC-KEY>
//
// other invalid combinations or extraneous items are ignored

func parseTag(s string) (*tagline, error) {

	t := &tagline{}

	countA := 0
	countC := 0
	countF := 0
	countI := 0
	countR := 0

words:
	for i, w := range strings.Split(strings.TrimSpace(s), " ") {

		if 0 == i {
			if _, ok := supportedTags[w]; ok {
				continue words
			}
			return nil, fault.ErrInvalidDnsTxtRecord
		}

		// ignore empty
		if "" == w {
			continue words
		}

		// require form: <letter>=<word>
		if len(w) < 3 || '=' != w[1] {
			return nil, fault.ErrInvalidDnsTxtRecord
		}

		// w[0]=tag character; w[1]= char('='); w[2:]=parameter
		parameter := w[2:]
		err := error(nil)
		switch w[0] {
		case 'a':
		addresses:
			for _, address := range strings.Split(parameter, ";") {
				if '[' == address[0] {
					end := len(address) - 1
					if ']' == address[end] {
						address = address[1:end]
					}
				}
				IP := net.ParseIP(address)
				if nil == IP {
					err = fault.ErrInvalidIpAddress
					break addresses
				} else {
					err = nil
					if nil != IP.To4() {
						t.ipv4 = IP
					} else {
						t.ipv6 = IP
					}
				}
			}
			countA += 1

		case 'c':
			t.connectPort, err = getPort(parameter)
			countC += 1
		case 's': // not actually used but stil check
			_, err = getPort(parameter)
		case 'r':
			t.rpcPort, err = getPort(parameter)
			countR += 1
		case 'i':
			t.peerID = parameter
			countI += 1
		case 'f':
			if len(parameter) != fingerprintLength {
				err = fault.ErrInvalidFingerprint
			} else {
				t.certificateFingerprint, err = hex.DecodeString(parameter)
				if nil != err {
					err = fault.ErrInvalidFingerprint
				}
			}
			countF += 1
		default:
			err = fault.ErrInvalidDnsTxtRecord
		}
		if nil != err {
			return nil, err
		}
	}

	// ensure that there is only one each of the required items
	if countA != 1 || countC != 1 || countF != 1 || countI != 1 || countR != 1 {
		return nil, fault.ErrInvalidDnsTxtRecord
	}

	return t, nil
}

func getPort(s string) (uint16, error) {

	port, err := strconv.Atoi(s)
	if nil != err {
		return 0, fault.ErrInvalidPortNumber
	}
	if port < 1 || port > 65535 {
		return 0, fault.ErrInvalidPortNumber
	}
	return uint16(port), nil
}
