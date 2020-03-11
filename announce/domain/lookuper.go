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

// Lookuper - interface to lookup DNS record
type Lookuper interface {
	Lookup(string) ([]DnsTXT, error)
}

type lookuper struct {
	log *logger.L
	f   func(string) ([]string, error)
}

// Lookup - query DNS TXT record
func (l *lookuper) Lookup(domainName string) ([]DnsTXT, error) {
	log := l.log
	var result []DnsTXT
	if "" == domainName {
		log.Error("invalid node domain")
		return result, fault.InvalidNodeDomain
	}

	txts, err := l.f(domainName)
	if nil != err {
		log.Errorf("lookup TXT record error: %s", err)
		return result, err
	}

	for i, t := range txts {
		t = strings.TrimSpace(t)
		txt, err := Parse(t)

		if nil != err {
			log.Debugf("ignore TXT[%d]: %q  error: %s", i, t, err)
		} else {
			log.Infof("process TXT[%d]: %q", i, t)
			log.Infof("result[%d]: IPv4: %q  IPv6: %q  rpc: %d  connect: %d", i, txt.IPv4, txt.IPv6, txt.RPCPort, txt.ConnectPort)
			log.Infof("result[%d]: peer public key: %x", i, txt.PublicKey)
			log.Infof("result[%d]: rpc fingerprint: %x", i, txt.CertificateFingerprint)

			if nil == txt.IPv4 && nil == txt.IPv6 {
				log.Debugf("result[%d]: ignoring invalid record", i)
			} else {
				result = append(result, *txt)
			}
		}
	}

	return result, nil
}

// NewLookuper - new Lookuper interface
func NewLookuper(log *logger.L, f func(string) ([]string, error)) Lookuper {
	return &lookuper{
		log: log,
		f:   f,
	}
}
