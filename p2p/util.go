package p2p

import (
	"net"
	"strconv"
	"strings"

	"github.com/bitmark-inc/bitmarkd/fault"
)

// ParseHostPort - parse host:port  return version(ip4/ip6), ip, port, error
func ParseHostPort(hostPort string) (string, string, string, error) {

	host, port, err := net.SplitHostPort(hostPort)
	if nil != err {
		return "", "", "", err
	}

	ip := strings.Trim(host, " ")
	if "*" == host {
		ip = "::"
	}
	numericPort, err := strconv.Atoi(strings.Trim(port, " "))
	if nil != err {
		return "", "", "", err
	}
	if numericPort < 1 || numericPort > 65535 {
		return "", "", "", fault.InvalidPortNumber
	}
	netIP := net.ParseIP(ip)
	var ver string
	if nil != netIP.To4() {
		ver = "ip4"
	} else {
		ver = "ip6"
	}

	return ver, ip, strconv.Itoa(numericPort), nil
}
