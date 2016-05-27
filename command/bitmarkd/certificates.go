// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"crypto/tls"
	//"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/certgen"
	"github.com/bitmark-inc/listener"
	"github.com/bitmark-inc/logger"
	"golang.org/x/crypto/sha3"
	"io/ioutil"
	"os"
	"time"
)

// Verify that a set of listener parameters are valid
// and return the certificate
func verifyListen(log *logger.L, name string, server *serverChannel) ([]byte, bool) {
	if server.limit < 0 {
		log.Errorf("invalid %s limit: %d", name, server.limit)
		return nil, false
	}

	// listening is disabled
	if 0 == server.limit || 0 == len(server.addresses) {
		server.limit = 0
		return nil, true
	}

	if !util.EnsureFileExists(server.certificateFileName) {
		log.Errorf("certificate: %q does not exist", server.certificateFileName)
		return nil, false
	}

	if !util.EnsureFileExists(server.keyFileName) {
		log.Errorf("private key: %q does not exist", server.keyFileName)
		return nil, false
	}

	// set up TLS
	keyPair, err := tls.LoadX509KeyPair(server.certificateFileName, server.keyFileName)
	if err != nil {
		log.Errorf("%s failed to load keypair: %v", name, err)
		return nil, false
	}

	server.tlsConfiguration = &tls.Config{
		Certificates: []tls.Certificate{
			keyPair,
		},
	}

	// fingerprint := CertificateFingerprint(keyPair.Certificate[0])
	// log.Infof("SHA3-256 fingerprint: %x", fingerprint)

	// store certificate
	//announce.AddCertificate(fingerprint, keyPair.Certificate[0]) // ***** FIX THIS: restore when ready

	// create limiter
	server.limiter = listener.NewLimiter(server.limit)

	return keyPair.Certificate[0], true
}

// create a self-signed certificate
func makeSelfSignedCertificate(name string, certificateFileName string, privateKeyFileName string, override bool, extraHosts []string) error {

	if util.EnsureFileExists(certificateFileName) {
		return fault.ErrCertificateFileAlreadyExists
	}

	if util.EnsureFileExists(privateKeyFileName) {
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

	if err = ioutil.WriteFile(privateKeyFileName, key, 0600); err != nil {
		os.Remove(certificateFileName)
		return err
	}

	return nil
}

// compute the fingerprint of a certificate
//
// FreeBSD: openssl x509 -outform DER -in bitmarkd-local-rpc.crt | sha3sum -a 256
func CertificateFingerprint(certificate []byte) [32]byte {
	return sha3.Sum256(certificate)
}
