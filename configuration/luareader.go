// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package configuration

import (
	"net"

	"github.com/yuin/gluamapper"
	lua "github.com/yuin/gopher-lua"
)

// ParseConfigurationFile - read and execute a Lua files and assign
// the results to a configuration structure
func ParseConfigurationFile(fileName string, config interface{}) error {
	L := lua.NewState()
	defer L.Close()

	L.OpenLibs()

	// create the global "arg" table
	// arg[0] = config file
	arg := &lua.LTable{}
	arg.Insert(0, lua.LString(fileName))
	L.SetGlobal("arg", arg)

	// prepare global "interface_public_ips" table
	addrList, err := net.InterfaceAddrs()
	if nil == err {

		// RFC 1918 (Address Allocation for Private Internets) [/8 /12 and /16]
		rfc1918_8 := net.ParseIP("10.0.0.0")
		rfc1918_12 := net.ParseIP("172.16.0.0")
		rfc1918_16 := net.ParseIP("192.168.0.0")

		// RFC 3927 (Dynamic Configuration of IPv4 Link-Local Addresses) [/16]
		rfc3927_16 := net.ParseIP("169.254.0.0")

		// RFC 4193 (Unique Local IPv6 Unicast Addresses) [/7]
		rfc4193_7 := net.ParseIP("fc00::")

		// table for the list of addresses
		addr := &lua.LTable{}
		j := 1 // lua indices start at 1
	ip_loop:
		for _, a := range addrList {
			ip, _, err := net.ParseCIDR(a.String())
			if nil != err || !ip.IsGlobalUnicast() {
				// exclude most non-routable addresses
				// like loopback and IPv6 link local
				continue ip_loop
			}

			if ip4 := ip.To4(); nil != ip4 {
				// mask to specific IPv4 network sizes
				ip4_8 := ip4.Mask(net.CIDRMask(8, 32))
				ip4_12 := ip4.Mask(net.CIDRMask(12, 32))
				ip4_16 := ip4.Mask(net.CIDRMask(16, 32))

				// check if IPv4 non-routable addresses
				if ip4_8.Equal(rfc1918_8) ||
					ip4_12.Equal(rfc1918_12) ||
					ip4_16.Equal(rfc1918_16) ||
					ip4_16.Equal(rfc3927_16) {
					continue ip_loop
				}

			} else { // IPv6

				// mask to specific IPv6 network sizes
				ip6_7 := ip.Mask(net.CIDRMask(7, 128))

				// check if IPv6 non-routable addresses
				if ip6_7.Equal(rfc4193_7) {
					continue ip_loop
				}
			}
			addr.Insert(j, lua.LString(ip.String()))
			j += 1
		}
		L.SetGlobal("interface_public_ips", addr)
	}

	// execute configuration
	if err := L.DoFile(fileName); err != nil {
		return err
	}

	mapperOption := gluamapper.Option{
		NameFunc: func(s string) string {
			return s
		},
		TagName: "gluamapper",
	}
	mapper := gluamapper.Mapper{Option: mapperOption}
	err = mapper.Map(L.Get(L.GetTop()).(*lua.LTable), config)
	return err
}
