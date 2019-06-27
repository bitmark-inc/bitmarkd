// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package announce

import (
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"

	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/logger"
)

const (
	timeInterval = 1 * time.Hour // time interval for re-fetching nodes domain
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

loop:
	for _, server := range conf.Servers {

		s := net.JoinHostPort(server, conf.Port)
		log.Debugf("dns resolver server: %q", s)
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

		for _, s := range sections {
			ttl := getTTL(s)
			if 0 < ttl {
				log.Infof("got TTL record: %d", ttl)
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
		return fault.ErrInvalidNodeDomain
	}

	texts, err := net.LookupTXT(domain)
	if nil != err {
		log.Errorf("lookup TXT record error: %s", err)
		return err
	}

	// process DNS entries
	for i, t := range texts {
		t = strings.TrimSpace(t)
		tag, err := parseTag(t)
		if nil != err {
			log.Debugf("ignore TXT[%d]: %q  error: %s", i, t, err)
		} else {
			log.Infof("process TXT[%d]: %q", i, t)
			log.Infof("result[%d]: IPv4: %q  IPv6: %q  rpc: %d  connect: %d", i, tag.ipv4, tag.ipv6, tag.rpcPort, tag.connectPort)
			log.Infof("result[%d]: peer public key: %x", i, tag.publicKey)
			log.Infof("result[%d]: rpc fingerprint: %x", i, tag.certificateFingerprint)

			listeners := []byte{}

			if nil != tag.ipv4 {
				c1 := util.ConnectionFromIPandPort(tag.ipv4, tag.connectPort)
				listeners = append(listeners, c1.Pack()...)
			}
			if nil != tag.ipv6 {
				c2 := util.ConnectionFromIPandPort(tag.ipv6, tag.connectPort)
				listeners = append(listeners, c2.Pack()...)
			}

			if nil == tag.ipv4 && nil == tag.ipv6 {
				log.Debugf("result[%d]: ignoring invalid record", i)
			} else {
				log.Infof("result[%d]: adding: %x", i, listeners)

				// internal add, as lock is already held
				addPeer(tag.publicKey, listeners, 0)
			}
		}
	}

	return nil
}
