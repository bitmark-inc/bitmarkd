// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package announce

import (
	"github.com/bitmark-inc/bitmarkd/util"
)

// add a certificate
func AddCertificate(fingerprint *util.FingerprintBytes, certificate []byte) {
	announce.certificatePool.Add(fingerprint[:], certificate)
}

// fetch a certificate
func GetCertificate(fingerprint *util.FingerprintBytes) ([]byte, bool) {
	certificate, found := announce.certificatePool.Get(fingerprint[:])
	return certificate, found
}
