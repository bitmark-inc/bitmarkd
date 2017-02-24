package announce

import (
	"encoding/hex"
	"github.com/bitmark-inc/bitmarkd/fault"
	"net"
	"strconv"
	"strings"
)

// the tag to detect applicable TXT records from DNS
const (
	taggedTXT         = "bitmark=v2"
	publicKeyLength   = 2 * 32 // characters
	fingerprintLength = 2 * 32 // characters
)

type tagline struct {
	ipv4                   net.IP
	ipv6                   net.IP
	rpcPort                uint16
	subscribePort          uint16
	connectPort            uint16
	certificateFingerprint []byte
	publicKey              []byte
}

// decode DNS TXT records of these forms
//
//   <TAG> a=<IPv4;IPv6> c=<PORT> s=<PORT> r=<PORT> f=<SHA3-256(cert)> p=<PUBLIC-KEY>
//
// other invalid combinations or extraneous items are ignored

func parseTag(s string) (*tagline, error) {

	t := &tagline{}

words:
	for i, w := range strings.Split(strings.TrimSpace(s), " ") {

		if 0 == i {
			if taggedTXT == w {
				continue
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
					err = fault.ErrInvalidIPAddress
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

		case 'c':
			t.connectPort, err = getPort(parameter)
		case 's':
			t.subscribePort, err = getPort(parameter)
		case 'r':
			t.rpcPort, err = getPort(parameter)
		case 'p':
			if len(parameter) != publicKeyLength {
				err = fault.ErrInvalidPublicKey
			} else {
				t.publicKey, err = hex.DecodeString(parameter)
				if nil != err {
					err = fault.ErrInvalidPublicKey
				}
			}
		case 'f':
			if len(parameter) != fingerprintLength {
				err = fault.ErrInvalidFingerprint
			} else {
				t.certificateFingerprint, err = hex.DecodeString(parameter)
				if nil != err {
					err = fault.ErrInvalidFingerprint
				}
			}
		default:
			err = fault.ErrInvalidDnsTxtRecord
		}
		if nil != err {
			return nil, err
		}
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
