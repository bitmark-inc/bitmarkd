// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package domain

import (
	"fmt"
	"net"
	"time"

	"github.com/bitmark-inc/bitmarkd/announce/receptor"

	"github.com/bitmark-inc/bitmarkd/announce/parameter"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/miekg/dns"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/logger"
)

const (
	loggerCategory = "domain"
	configFile     = "/etc/resolv.conf"
)

type Domain interface {
	background.Process
}

type domain struct {
	logger    *logger.L
	name      string
	lookuper  Lookuper
	receptors receptor.Receptor
}

func NewDomain(nodesDomain string, receptors receptor.Receptor, f func(string) ([]string, error)) (Domain, error) {
	d := &domain{
		logger:    logger.New(loggerCategory),
		name:      nodesDomain,
		lookuper:  NewLookuper(nodesDomain),
		receptors: receptors,
	}

	d.logger.Info("initialising…")

	txts, err := d.lookuper.Lookup(f)
	if nil != err {
		return nil, err
	}

	addTxts(txts, d.logger, d.receptors)
	return d, nil
}

func (d domain) Run(_ interface{}, shutdown <-chan struct{}) {
	timer := time.After(intervalTime(d.name, d.logger))
	log := d.logger

loop:
	for {
		select {
		case <-timer:
			timer = time.After(intervalTime(d.name, d.logger))
			txts, err := d.lookuper.Lookup(net.LookupTXT)
			if nil != err {
				log.Errorf("domain name lookup with error: %s", err)
				continue
			}
			addTxts(txts, d.logger, d.receptors)

		case <-shutdown:
			break loop
		}
	}
}

// get interval time for lookup node domain txt record
func intervalTime(domain string, log *logger.L) time.Duration {
	t := parameter.ReFetchingInterval
	var servers []string // dns name server

	// reading default configuration file
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
				if parameter.ReFetchingInterval > ttlSecond {
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

func addTxts(txts []DnsTxt, log *logger.L, receptors receptor.Receptor) {
	// TODO: move this logic into addPeer
	for i, txt := range txts {
		var listeners []ma.Multiaddr
		if nil != txt.IPv4 {
			ipv4ma, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%v/%s/%s", txt.IPv4, txt.ConnectPort, parameter.Protocol, txt.PeerID))
			if nil == err {
				listeners = append(listeners, ipv4ma)
			} else {
				log.Warnf("form ipv6 ma error :%v", err)
			}
		}
		if nil != txt.IPv6 {
			ipv6ma, err := ma.NewMultiaddr(fmt.Sprintf("/ip6/%s/tcp/%v/%s/%s", txt.IPv6, txt.ConnectPort, parameter.Protocol, txt.PeerID))
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
		receptors.Add(id, listeners, uint64(time.Now().Unix()))
		receptors.Tree().Print(false)
	}
}
