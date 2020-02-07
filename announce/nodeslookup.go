// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package announce

import (
	"fmt"
	"github.com/bitmark-inc/bitmarkd/announce/domain"
	"net"
	"time"

	"github.com/bitmark-inc/logger"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/miekg/dns"
	ma "github.com/multiformats/go-multiaddr"
)

const (
	reFetchingInterval = 1 * time.Hour // re-fetching nodes domain
	nodeProtocol       = "p2p"
	loggerCategory     = "nodeslookup"
)

type lookup struct {
	logger     *logger.L
	domainName string
	lookuper   domain.Lookuper
}

func (l *lookup) initialise(nodesDomain string) error {
	l.logger = logger.New(loggerCategory)
	l.logger.Info("initialising…")
	l.domainName = nodesDomain
	l.lookuper = domain.NewLookuper(nodesDomain)

	txts, err := l.lookuper.Lookup(net.LookupTXT)
	if nil != err {
		return err
	}
	addTxts(txts, l.logger)
	return nil
}

func (l *lookup) Run(_ interface{}, shutdown <-chan struct{}) {
	timer := time.After(intervalTime(l.domainName, l.logger))
	log := l.logger

loop:
	for {
		select {
		case <-timer:
			timer = time.After(intervalTime(l.domainName, l.logger))
			txts, err := l.lookuper.Lookup(net.LookupTXT)
			if nil != err {
				log.Errorf("domain name lookup with error: %s", err)
				continue
			}
			addTxts(txts, l.logger)

		case <-shutdown:
			break loop
		}
	}
}

// get interval time for lookup node domain txt record
func intervalTime(domain string, log *logger.L) time.Duration {
	t := reFetchingInterval
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
	// limit the name servers to lookup
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
			ttl := ttl(section)
			if 0 < ttl {
				log.Infof("got TTL record from server %q value %d", s, ttl)
				ttlSecond := time.Duration(ttl) * time.Second
				if reFetchingInterval > ttlSecond {
					t = ttlSecond
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
func ttl(rrs []dns.RR) uint32 {
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

func addTxts(txts []domain.DnsTxt, log *logger.L) {
	// TODO: move this logic into addPeer
	for i, txt := range txts {
		var listeners []ma.Multiaddr
		if nil != txt.IPv4 {
			ipv4ma, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%v/%s/%s", txt.IPv4, txt.ConnectPort, nodeProtocol, txt.PeerID))
			if nil == err {
				listeners = append(listeners, ipv4ma)
			} else {
				log.Warnf("form ipv6 ma error :%v", err)
			}
		}
		if nil != txt.IPv6 {
			ipv6ma, err := ma.NewMultiaddr(fmt.Sprintf("/ip6/%s/tcp/%v/%s/%s", txt.IPv6, txt.ConnectPort, nodeProtocol, txt.PeerID))
			if nil == err {
				listeners = append(listeners, ipv6ma)
			} else {
				log.Warnf("form ipv6 ma error :%v", err)
			}
		}

		id, err := peer.IDB58Decode(txt.PeerID)
		if err != nil {
			log.Warnf("ID DecodeBase58 Error :%v ID::%v", err, txt.PeerID)
			continue
		}
		log.Infof("result[%d]: adding id:%s", i, txt.PeerID)
		addPeer(id, listeners, uint64(time.Now().Unix()))
		globalData.peerTree.Print(false)
	}
}
