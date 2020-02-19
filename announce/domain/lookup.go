// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package domain

import (
	"strings"

	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/logger"
)

// Lookuper - interface to lookup domain name
type Lookuper interface {
	Lookup(func(string) ([]string, error)) ([]DnsTxt, error)
}

type lookuper struct {
	logger *logger.L
	domain string
}

func NewLookuper(domain string, log *logger.L) Lookuper {
	return &lookuper{
		logger: log,
		domain: domain,
	}
}

// lookup node domain for the peering
func (l *lookuper) Lookup(f func(string) ([]string, error)) ([]DnsTxt, error) {
	log := l.logger
	if "" == l.domain {
		return nil, fault.InvalidNodeDomain
	}

	texts, err := f(l.domain)
	if nil != err {
		return nil, err
	}

	result := make([]DnsTxt, 0)
loop:
	// process DNS entries
	for i, t := range texts {
		t = strings.TrimSpace(t)
		tag, err := parseTxt(t)
		if nil != err {
			log.Debugf("ignore TXT[%d]: %q  error: %s", i, t, err)
			return nil, err
		} else {
			log.Infof("process TXT[%d]: %q", i, t)
			log.Infof("result[%d]: IPv4: %q  IPv6: %q  rpc: %d  connect: %d", i, tag.IPv4, tag.IPv6, tag.RpcPort, tag.ConnectPort)
			log.Infof("result[%d]: peer ID: %s", i, tag.PeerID)
			log.Infof("result[%d]: rpc fingerprint: %x", i, tag.CertificateFingerprint)
			if nil == tag.IPv4 && nil == tag.IPv6 {
				log.Debugf("result[%d]: ignoring invalid record", i)
				break loop
			}

			result = append(result, *tag)
		}
	}
	return result, nil
}
