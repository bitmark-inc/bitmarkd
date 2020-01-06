// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package announce

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/logger"
	peerlib "github.com/libp2p/go-libp2p-core/peer"
	"github.com/miekg/dns"
	ma "github.com/multiformats/go-multiaddr"
)

const (
	timeInterval = 1 * time.Hour // time interval for re-fetching nodes domain
	nodeProtocol = "p2p"
)

type nodesLookup struct {
	logger *logger.L

	nodesDomain string
}

func (n *nodesLookup) initialise(nodesDomain string) error {

	n.logger = logger.New("nodeslookup")
	n.logger.Info("initialising…")
	n.nodesDomain = nodesDomain

	return lookupNodesDomain(n.nodesDomain, n.logger)
}

func (n *nodesLookup) Run(args interface{}, shutdown <-chan struct{}) {

	timer := time.After(getIntervalTime(n.nodesDomain, n.logger))

loop:
	for {
		select {
		case <-timer:
			timer = time.After(getIntervalTime(n.nodesDomain, n.logger))
			lookupNodesDomain(n.nodesDomain, n.logger)

		case <-shutdown:
			break loop
		}
	}
}

// get interval time for lookup node domain txt record
func getIntervalTime(domain string, log *logger.L) time.Duration {

	t := timeInterval
	var servers []string // dns name server

	// reading default configuration file
	const configFile = "/etc/resolv.conf"
	conf, err := dns.ClientConfigFromFile(configFile)

	if nil != err {
		log.Warnf("reading %s error: %s", configFile, err)
		goto done
	}

	if 0 == len(conf.Servers) {
		log.Warnf("cannot get dns name server")
		goto done
	}

	servers = conf.Servers
	// limit the nameservers to lookup
	// https://www.freebsd.org/cgi/man.cgi?resolv.conf
	if len(servers) > 3 {
		servers = servers[:3]
	}

loop:
	for _, server := range servers {

		s := net.JoinHostPort(server, conf.Port)
		c := dns.Client{}
		msg := dns.Msg{}
		msg.SetQuestion(domain+".", dns.TypeSOA) // fixed for type SOA

		r, _, err := c.Exchange(&msg, s)
		if nil != err {
			log.Debugf("exchange with dns server %q error: %s", s, err)
			continue loop
		}

		if 0 == len(r.Ns) && 0 == len(r.Answer) && 0 == len(r.Extra) {
			log.Debugf("no resource record found by dns server %q", s)
			continue loop
		}

		sections := [][]dns.RR{r.Answer, r.Ns, r.Extra}

		for _, section := range sections {
			ttl := getTTL(section)
			if 0 < ttl {
				log.Infof("got TTL record from server %q value %d", s, ttl)
				ttlSec := time.Duration(ttl) * time.Second
				if timeInterval > ttlSec {
					t = ttlSec
					break loop
				}
			}
		}
	}

done:
	log.Infof("time to re-fetching node domain: %v", t)
	return t
}

// get TTL record from a resource record
func getTTL(rrs []dns.RR) uint32 {
	if 0 == len(rrs) {
		return 0
	}
	for _, rr := range rrs {
		if soa, ok := rr.(*dns.SOA); ok {
			return soa.Hdr.Ttl
		} else {
			return rr.Header().Ttl
		}
	}
	return 0
}

// lookup node domain for the peering
func lookupNodesDomain(domain string, log *logger.L) error {

	if "" == domain {
		log.Error("invalid node domain")
		return fault.InvalidNodeDomain
	}

	texts, err := net.LookupTXT(domain)
	if nil != err {
		log.Errorf("lookup TXT record error: %s", err)
		return err
	}
loop:
	// process DNS entries
	for i, t := range texts {
		t = strings.TrimSpace(t)
		tag, err := parseTag(t)
		if nil != err {
			log.Debugf("ignore TXT[%d]: %q  error: %s", i, t, err)
		} else {
			log.Infof("process TXT[%d]: %q", i, t)
			log.Infof("result[%d]: IPv4: %q  IPv6: %q  rpc: %d  connect: %d", i, tag.ipv4, tag.ipv6, tag.rpcPort, tag.connectPort)
			log.Infof("result[%d]: peer ID: %s", i, tag.peerID)
			log.Infof("result[%d]: rpc fingerprint: %x", i, tag.certificateFingerprint)
			if nil == tag.ipv4 && nil == tag.ipv6 {
				log.Debugf("result[%d]: ignoring invalid record", i)
				break
			}
			var listeners []ma.Multiaddr
			if nil != tag.ipv4 {
				ipv4ma, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%v/%s/%s", tag.ipv4, tag.connectPort, nodeProtocol, tag.peerID))
				if nil == err {
					listeners = append(listeners, ipv4ma)
				} else {
					log.Warnf("form ipv6 ma error :%v", err)
				}
			}
			if nil != tag.ipv6 {
				ipv6ma, err := ma.NewMultiaddr(fmt.Sprintf("/ip6/%s/tcp/%v/%s/%s", tag.ipv6, tag.connectPort, nodeProtocol, tag.peerID))
				if nil == err {
					listeners = append(listeners, ipv6ma)
				} else {
					log.Warnf("form ipv6 ma error :%v", err)
				}
			}

			id, err := peerlib.IDB58Decode(tag.peerID)
			if err != nil {
				log.Warnf("ID DecodeBase58 Error :%v ID::%v", err, tag.peerID)
				continue loop
			}
			log.Infof("result[%d]: adding id:%s", i, tag.peerID)
			addPeer(id, listeners, uint64(time.Now().Unix()))
			globalData.peerTree.Print(false)
		}
	}
	return nil
}
