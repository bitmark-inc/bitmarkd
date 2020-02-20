package util

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	peerlib "github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/bitmark-inc/bitmarkd/fault"
)

// IDCompare The result will be 0 if a==b, -1 if a < b, and +1 if a > b.
func IDCompare(ida, idb peerlib.ID) int {
	return strings.Compare(ida.String(), idb.String())
}

// IDEqual if 2 peer id are equal
func IDEqual(ida, idb peerlib.ID) bool {
	return ida.String() == idb.String()
}

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

//DualStackAddrToIPV4IPV6 read ip:port list and make dualstack address "*" into 0.0.0.0:port and [::]:port.
// If there any of 0.0.0.0 or [::] is also in the given list, they  will merge to one.
func DualStackAddrToIPV4IPV6(ipPorts []string) (iPPorts []string) {
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

// MaAddrToAddrInfo Convert  multiAddr to peer.AddrInfo; Must Include  ID
func MaAddrToAddrInfo(maAddr ma.Multiaddr) (*peerlib.AddrInfo, error) {
	info, err := peerlib.AddrInfoFromP2pAddr(maAddr)
	if err != nil {
		return nil, err
	}
	if nil == info {
		return nil, fault.AddrinfoIsNil
	}
	return info, nil
}

// MaAddrsToAddrInfos Convert  []multiAddr to []peer.AddrInfo
func MaAddrsToAddrInfos(maAddrs []ma.Multiaddr) ([]peerlib.AddrInfo, error) {
	if len(maAddrs) < 1 {
		return nil, fault.NoAddress
	}
	infos, err := peerlib.AddrInfosFromP2pAddrs(maAddrs...)
	if err != nil {
		return nil, err
	}
	if nil == infos {
		return nil, fault.AddrinfoIsNil
	}
	return infos, nil
}

// GetMultiAddrsFromBytes take  [][]byte listeners and convert them into []Multiaddr format
func GetMultiAddrsFromBytes(listners [][]byte) []ma.Multiaddr {
	var maAddrs []ma.Multiaddr
	for _, addr := range listners {
		maAddr, err := ma.NewMultiaddrBytes(addr)
		if nil == err {
			maAddrs = append(maAddrs, maAddr)
		}
	}
	return maAddrs
}

//GetBytesFromMultiaddr take []Multiaddr format listeners and convert them into   [][]byte
func GetBytesFromMultiaddr(listners []ma.Multiaddr) [][]byte {
	var byteAddrs [][]byte
	for _, addr := range listners {
		byteAddrs = append(byteAddrs, addr.Bytes())
	}
	return byteAddrs
}

// MaAddrToString take an [][]byte and convert them it to multiAddress and return its presented string
func MaAddrToString(maAddrs []ma.Multiaddr) []string {
	var addrsStr []string
	for _, addr := range maAddrs {
		addrsStr = append(addrsStr, addr.String())
	}
	return addrsStr
}

// ByteAddrsToString take an [][]byte and convert them it to multiAddress and return its presented string
func ByteAddrsToString(addrs [][]byte) []string {
	var addrsStr []string
	for _, addr := range addrs {
		newAddr, err := ma.NewMultiaddrBytes(addr)
		if nil == err {
			addrsStr = append(addrsStr, newAddr.String())
		}
	}
	return addrsStr
}

//IsMultiAddrIPV4 check if an ipv4 address
func IsMultiAddrIPV4(addr ma.Multiaddr) bool {
	for _, protocol := range addr.Protocols() {
		if protocol.Name == "ip4" {
			return true
		}
	}
	return false
}

//IsMultiAddrIPV6 check if an ipv4 address
func IsMultiAddrIPV6(addr ma.Multiaddr) bool {
	for _, protocol := range addr.Protocols() {
		if protocol.Name == "ip6" {
			return true
		}
	}
	return false
}

// PrintMaAddrs print out all ma with a new line seperater
func PrintMaAddrs(addrs []ma.Multiaddr) string {
	var stringAddr string
	for _, addr := range addrs {
		stringAddr = fmt.Sprintf("%s%s\n", stringAddr, addr.String())
	}
	return stringAddr
}
