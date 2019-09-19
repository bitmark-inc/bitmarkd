package p2p

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

func announceMuxAddr(ipportAnnounce []ma.Multiaddr, protocol string, id peer.ID) []ma.Multiaddr {
	maAddrs := []ma.Multiaddr{}
	p2pMa, err := ma.NewMultiaddr(fmt.Sprintf("/%s/%s", protocol, id.String()))
	if err != nil {
		return nil
	}
	for _, addr := range ipportAnnounce {
		maAddr := addr.Encapsulate(p2pMa)
		maAddrs = append(maAddrs, maAddr)
	}
	return maAddrs
}

// IPPortToMultiAddr generate a multiaddr from input array of listening address
func IPPortToMultiAddr(addrsStr []string) []ma.Multiaddr {
	var maAddrs []ma.Multiaddr
loop:
	for _, IPPort := range addrsStr {
		ver, ip, port, err := ParseHostPort(IPPort)
		if err != nil {
			continue loop
		}
		addr, err := ma.NewMultiaddr(fmt.Sprintf("/%s/%s/tcp/%s", ver, ip, port))
		if err != nil {
			continue loop
		}
		maAddrs = append(maAddrs, addr)
	}
	return maAddrs
}

func parseIPPort(hostPort string) (v string, ip string, port uint16, err error) {
	host, portStr, err := net.SplitHostPort(hostPort)
	if nil != err {
		return "", "", 0, fault.ErrInvalidIpAddress
	}

	IP := net.ParseIP(strings.Trim(host, " "))
	if nil == IP {
		return "", "", 0, fault.ErrInvalidIpAddress
	}
	if nil != IP.To4() {
		v = "ipv4"
	} else {
		v = "ipv6"
	}

	numericPort, err := strconv.Atoi(strings.Trim(portStr, " "))
	if nil != err {
		return "", "", 0, err
	}
	if numericPort < 1 || numericPort > 65535 {
		return "", "", 0, fault.ErrInvalidPortNumber
	}
	return v, IP.String(), uint16(numericPort), nil
}
