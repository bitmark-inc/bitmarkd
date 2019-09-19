package main

import (
	"fmt"

	ma "github.com/multiformats/go-multiaddr"
)

// NewListenMultiAddr generate a multiaddr from input array of listening address
func NewListenMultiAddr(listenAddr []string) (ma.Multiaddr, error) {
	var hostListenAddress ma.Multiaddr
	// TODO:  ipv6
	for idx, IPPort := range listenAddr {
		ip, port, err := ParseHostPort(IPPort)
		if err != nil {
			panic(err)
		}
		if 0 == idx {
			hostListenAddress, err = ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%s", ip, port))
			if err != nil {
				panic(err)
			}
			return hostListenAddress, err
		} else {
			additionAddress, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%s", ip, port))
			if err != nil {
				globalData.log.Warn(err.Error())
				break
			}
			hostListenAddress = hostListenAddress.Encapsulate(additionAddress)
		}
	}
	return hostListenAddress, nil
}
