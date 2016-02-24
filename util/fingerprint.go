// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package util

import (
	"crypto/sha256"
)

// to hold type for fingerprint
type FingerprintBytes [sha256.Size]byte

// fingerprint a certificate
//
// the fingerprint is SHA256 because of:
// openssl x509 -noout -in ~/.config/bitmarkd/bitmarkd.crt -text -fingerprint -sha256
// so this provides a way to double check on the command line
func Fingerprint(certificate []byte) FingerprintBytes {
	return sha256.Sum256(certificate)
}
