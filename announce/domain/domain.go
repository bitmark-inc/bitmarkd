// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package domain

import (
	"net"
	"time"

	"github.com/miekg/dns"

	"github.com/bitmark-inc/bitmarkd/announce/receptor"
	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/logger"
)

// startup node connection information is provided through DNS TXT records.
// The format:
// txt-record=a.b.c,"bitmark=v3 a=127.0.0.1;[::1] c=22136 r=22130 f=xxx p=xxx
// This package is to get DNS TXT record and parse it

const (
	timeInterval = 1 * time.Hour // time interval for re-fetching nodes domain
	configFile   = "/etc/resolv.conf"
)

type domain struct {
	log        *logger.L
	domainName string
	receptors  receptor.Receptor
	lookuper   Lookuper
}

// Run - background processing interface
func (d *domain) Run(_ interface{}, shutdown <-chan struct{}) {
	timer := time.After(interval(d.domainName, d.log))

loop:
	for {
		select {
		case <-timer:
			timer = time.After(interval(d.domainName, d.log))
			txts, err := d.lookuper.Lookup(d.domainName)
			if nil != err {
				continue loop
			}

			addTXTs(txts, d.log, d.receptors)

		case <-shutdown:
			break loop
		}
	}
}

// get interval time for lookup node domain txt record
func interval(domain string, log *logger.L) time.Duration {
	t := timeInterval
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
			ttl := ttl(section)
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

// New - return background processing interface
func New(log *logger.L, domainName string, receptors receptor.Receptor, f func(string) ([]string, error)) (background.Process, error) {
	log.Info("initialising…")

	d := &domain{
		log:        log,
		domainName: domainName,
		receptors:  receptors,
		lookuper:   NewLookuper(log, f),
	}

	txts, err := d.lookuper.Lookup(d.domainName)
	if nil != err {
		return nil, err
	}

	addTXTs(txts, log, receptors)

	return d, nil
}

func addTXTs(txts []DnsTXT, log *logger.L, receptors receptor.Receptor) {
	for i, t := range txts {
		var listeners []byte

		if nil != t.IPv4 {
			c1 := util.ConnectionFromIPandPort(t.IPv4, t.ConnectPort)
			listeners = append(listeners, c1.Pack()...)
		}
		if nil != t.IPv6 {
			c2 := util.ConnectionFromIPandPort(t.IPv6, t.ConnectPort)
			listeners = append(listeners, c2.Pack()...)
		}

		if nil == t.IPv4 && nil == t.IPv6 {
			log.Debugf("result[%d]: ignoring invalid record", i)
		} else {
			log.Infof("result[%d]: adding: %x", i, listeners)

			receptors.Add(t.PublicKey, listeners, uint64(time.Now().Unix()))
		}
	}
}
