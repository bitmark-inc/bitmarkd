// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.
// the tag to detect applicable TXT records from DNS

package domain

import (
	"encoding/hex"
	"github.com/bitmark-inc/bitmarkd/fault"
	"net"
	"strconv"
	"strings"
)

var supportedTags = map[string]struct{}{
	"bitmark-p2p=v1": {},
}

const (
	fingerprintLength = 2 * 32 // characters
	p2pIdentityLength = 52     // from host.ID().Pretty()
)

type DnsTxt struct {
	IPv4                   net.IP
	IPv6                   net.IP
	RpcPort                uint16
	ConnectPort            uint16
	CertificateFingerprint []byte
	PeerID                 string
}

// decode DNS TXT records of these forms
//
//   <TAG> a=<IPv4;IPv6> c=<PORT> r=<PORT> f=<SHA3-256(cert)> i=<PEER-ID>
//
// other invalid combinations or extraneous items are ignored

func parseTxt(s string) (*DnsTxt, error) {

	t := &DnsTxt{}

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
			return nil, fault.InvalidDnsTxtRecord
		}

		// ignore empty
		if "" == w {
			continue words
		}

		// require form: <letter>=<word>
		if len(w) < 3 || '=' != w[1] {
			return nil, fault.InvalidDnsTxtRecord
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
					err = fault.InvalidIpAddress
					break addresses
				} else {
					err = nil
					if nil != IP.To4() {
						t.IPv4 = IP
					} else {
						t.IPv6 = IP
					}
				}
			}
			countA += 1

		case 'c':
			t.ConnectPort, err = getPort(parameter)
			countC += 1
		case 'r':
			t.RpcPort, err = getPort(parameter)
			countR += 1
		case 'i':
			if len(parameter) != p2pIdentityLength {
				err = fault.InvalidIdentityName
			} else {
				t.PeerID = parameter
			}
			countI += 1
		case 'f':
			if len(parameter) != fingerprintLength {
				err = fault.InvalidFingerprint
			} else {
				t.CertificateFingerprint, err = hex.DecodeString(parameter)
				if nil != err {
					err = fault.InvalidFingerprint
				}
			}
			countF += 1
		default:
			err = fault.InvalidDnsTxtRecord
		}
		if nil != err {
			return nil, err
		}
	}

	// ensure that there is only one each of the required items
	if countA != 1 || countC != 1 || countF != 1 || countI != 1 || countR != 1 {
		return nil, fault.InvalidDnsTxtRecord
	}

	return t, nil
}

func getPort(s string) (uint16, error) {

	port, err := strconv.Atoi(s)
	if nil != err {
		return 0, fault.InvalidPortNumber
	}
	if port < 1 || port > 65535 {
		return 0, fault.InvalidPortNumber
	}
	return uint16(port), nil
}
