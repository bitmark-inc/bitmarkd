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

func makeDualStackAddrs(ipPorts []string) (iPPorts []string) {
	uniqIPs := make(map[string]bool)
	for _, ipPort := range ipPorts {
		sep := strings.Split(ipPort, ":")
		if len(sep) == 2 && "*" == sep[0] {
			ipv4 := "0.0.0.0:" + sep[1]
			ipv6 := "[::]:" + sep[1]
			uniqIPs[ipv4] = true
			uniqIPs[ipv6] = true
		} else {
			uniqIPs[ipPort] = true
		}
	}
	for key := range uniqIPs {
		iPPorts = append(iPPorts, key)
	}
	return
}
