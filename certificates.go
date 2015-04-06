// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"crypto/tls"
	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/configuration"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/certgen"
	"github.com/bitmark-inc/listener"
	"github.com/bitmark-inc/logger"
	"io/ioutil"
	"os"
	"time"
)

type verifiedListener struct {
	tlsConfiguration *tls.Config
	limiter          *listener.Limiter
}

// Verify that a set of listener parameters are valid
func verifyListen(log *logger.L, name string, server *serverChannel) (*util.FingerprintBytes, bool) {
	if server.limit < 0 {
		log.Errorf("invalid %s limit: %d", name, server.limit)
		return nil, false
	}

	// listening is disabled
	if 0 == server.limit || 0 == len(server.addresses) {
		server.limit = 0
		return nil, true
	}

	certificateFileName, exists := configuration.ResolveFileName(server.certificateFileName)
	if !exists {
		log.Errorf("certificate: does not exist: in '%s' or '%s'", certificateFileName, certificateFileName)
		return nil, false
	}

	keyFileName, exists := configuration.ResolveFileName(server.keyFileName)
	if !exists {
		log.Errorf("key: does not exist: in '%s' or '%s'", keyFileName, keyFileName)
		return nil, false
	}

	// set up TLS
	keyPair, err := tls.LoadX509KeyPair(certificateFileName, keyFileName)
	if err != nil {
		log.Errorf("%s failed to load keypair: %v", name, err)
		return nil, false
	}

	server.tlsConfiguration = &tls.Config{
		Certificates: []tls.Certificate{
			keyPair,
		},
	}

	fingerprint := util.Fingerprint(keyPair.Certificate[0])
	log.Infof("fingerprint = %x", fingerprint)

	// store certificate
	announce.AddCertificate(&fingerprint, keyPair.Certificate[0])

	// create limiter
	server.limiter = listener.NewLimiter(server.limit)

	return &fingerprint, true
}

// create a self-signed certificate
func makeSelfSignedCertificate(name string, certificateFileName string, keyFileName string, override bool, extraHosts []string) error {
	certificateFileName, exists := configuration.ResolveFileName(certificateFileName)
	if exists {
		return fault.ErrCertificateFileAlreadyExists
	}

	keyFileName, exists = configuration.ResolveFileName(keyFileName)
	if exists {
		return fault.ErrKeyFileAlreadyExists
	}

	org := "bitmarkd self signed cert for: " + name
	validUntil := time.Now().Add(10 * 365 * 24 * time.Hour)
	cert, key, err := certgen.NewTLSCertPair(org, validUntil, override, extraHosts)
	if err != nil {
		return err
	}

	if err = ioutil.WriteFile(certificateFileName, cert, 0666); err != nil {
		return err
	}

	if err = ioutil.WriteFile(keyFileName, key, 0600); err != nil {
		os.Remove(certificateFileName)
		return err
	}

	return nil
}
