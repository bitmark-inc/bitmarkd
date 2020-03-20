// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package certificate

import (
	"crypto/tls"

	"golang.org/x/crypto/sha3"

	"github.com/bitmark-inc/logger"
)

// Verify that a set of listener parameters are valid
// and return the certificate
func Get(log *logger.L, name, certificate, key string) (*tls.Config, [32]byte, error) {
	var fin [32]byte

	keyPair, err := tls.X509KeyPair([]byte(certificate), []byte(key))
	if err != nil {
		log.Errorf("%s failed to load keypair: %v", name, err)
		return nil, fin, err
	}

	tlsConfiguration := &tls.Config{
		Certificates: []tls.Certificate{
			keyPair,
		},
	}

	fin = fingerprint(keyPair.Certificate[0])

	return tlsConfiguration, fin, nil
}

// fingerprint - compute the fingerprint of a certificate
//
// FreeBSD: openssl x509 -outform DER -in bitmarkd-local-rpc.crt | sha3sum -a 256
func fingerprint(certificate []byte) [32]byte {
	return sha3.Sum256(certificate)
}
