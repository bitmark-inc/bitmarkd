// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package domain

import (
	"encoding/hex"
	"net"
	"strconv"
	"strings"

	"github.com/bitmark-inc/bitmarkd/fault"
)

// supported tag of TXT records from DNS
var supported = map[string]struct{}{
	"bitmark=v2": {},
	"bitmark=v3": {},
}

const (
	publicKeyLength   = 2 * 32 // characters
	fingerprintLength = 2 * 32 // characters
	maxPortNumber     = 65535
	minPortNumber     = 1
)

// DnsTXT - structure for dns txt record
type DnsTXT struct {
	IPv4                   net.IP
	IPv6                   net.IP
	RPCPort                uint16
	ConnectPort            uint16
	CertificateFingerprint []byte
	PublicKey              []byte
}

// Parse - parse a dns txt record
func Parse(s string) (*DnsTXT, error) {
	t := &DnsTXT{}

	countA := 0
	countC := 0
	countF := 0
	countP := 0
	countR := 0

words:
	for i, w := range strings.Split(strings.TrimSpace(s), " ") {

		if i == 0 {
			if _, ok := supported[w]; ok {
				continue words
			}
			return nil, fault.InvalidDnsTxtRecord
		}

		// ignore empty
		if w == "" {
			continue words
		}

		// require form: <letter>=<word>
		if len(w) < 3 || w[1] != '=' {
			return nil, fault.InvalidDnsTxtRecord
		}

		// w[0]=tag character; w[1]= char('='); w[2:]=parameter
		parameter := w[2:]
		err := error(nil)
		switch w[0] {
		case 'a':
		addresses:
			for _, address := range strings.Split(parameter, ";") {
				if address[0] == '[' {
					end := len(address) - 1
					if address[end] == ']' {
						address = address[1:end]
					}
				}
				IP := net.ParseIP(address)
				if IP == nil {
					err = fault.InvalidIpAddress
					break addresses
				} else {
					err = nil
					if IP.To4() != nil {
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
		case 's': // not actually used but still check
			_, err = getPort(parameter)
		case 'r':
			t.RPCPort, err = getPort(parameter)
			countR += 1
		case 'p':
			if len(parameter) != publicKeyLength {
				err = fault.InvalidPublicKey
			} else {
				t.PublicKey, err = hex.DecodeString(parameter)
				if err != nil {
					err = fault.InvalidPublicKey
				}
			}
			countP += 1
		case 'f':
			if len(parameter) != fingerprintLength {
				err = fault.InvalidFingerprint
			} else {
				t.CertificateFingerprint, err = hex.DecodeString(parameter)
				if err != nil {
					err = fault.InvalidFingerprint
				}
			}
			countF += 1
		default:
			err = fault.InvalidDnsTxtRecord
		}
		if err != nil {
			return nil, err
		}
	}

	// ensure that there is only one each of the required items
	if countA != 1 || countC != 1 || countF != 1 || countP != 1 || countR != 1 {
		return nil, fault.InvalidDnsTxtRecord
	}

	return t, nil
}

func getPort(s string) (uint16, error) {
	port, err := strconv.Atoi(s)
	if err != nil {
		return 0, fault.InvalidPortNumber
	}
	if port < minPortNumber || port > maxPortNumber {
		return 0, fault.InvalidPortNumber
	}
	return uint16(port), nil
}
