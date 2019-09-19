package util

import (
	"errors"
	"fmt"
	"strings"

	peerlib "github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/prometheus/common/log"
)

// IDCompare The result will be 0 if a==b, -1 if a < b, and +1 if a > b.
func IDCompare(ida, idb peerlib.ID) int {
	return strings.Compare(ida.String(), idb.String())
}

// IDEqual if 2 peer id are equal
func IDEqual(ida, idb peerlib.ID) bool {
	if ida.String() == idb.String() {
		return true
	}
	return false
}

// MaAddrToAddrInfo Convert  multiAddr to peer.AddrInfo
func MaAddrToAddrInfo(ma ma.Multiaddr) (*peerlib.AddrInfo, error) {
	info, err := peerlib.AddrInfoFromP2pAddr(ma)
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}
	if nil == info {
		return nil, errors.New("AddrInfo is nil")
	}
	//p.PeersRemote.AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)
	return info, nil
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

// GetBytesFromMultiaddr take []Multiaddr format listeners and convert them into   [][]byte
func GetBytesFromMultiaddr(listners []ma.Multiaddr) [][]byte {
	var byteAddrs [][]byte
	for _, addr := range listners {
		byteAddrs = append(byteAddrs, addr.Bytes())
	}
	return byteAddrs
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

// PrintMaAddrs print out all ma with a new line seperater
func PrintMaAddrs(addrs []ma.Multiaddr) string {
	var stringAddr string
	for _, addr := range addrs {
		stringAddr = fmt.Sprintf("%s%s\n", stringAddr, addr.String())
	}
	return stringAddr
}
