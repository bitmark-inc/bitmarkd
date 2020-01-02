// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"io/ioutil"
	"os"
	"time"

	"golang.org/x/crypto/sha3"

	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/certgen"
)

// create a self-signed certificate
func makeSelfSignedCertificate(name string, certificateFileName string, privateKeyFileName string, override bool, extraHosts []string) error {

	if util.EnsureFileExists(certificateFileName) {
		return fault.CertificateFileAlreadyExists
	}

	if util.EnsureFileExists(privateKeyFileName) {
		return fault.KeyFileAlreadyExists
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

// CertificateFingerprint - compute the fingerprint of a certificate
//
// FreeBSD: openssl x509 -outform DER -in bitmarkd-local-rpc.crt | sha3sum -a 256
// Darwin:  openssl x509 -outform DER -in bitmarkd-local-rpc.crt | sha3-256sum
func CertificateFingerprint(certificate []byte) [32]byte {
	return sha3.Sum256(certificate)
}
