package p2p

import (
	"fmt"

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
