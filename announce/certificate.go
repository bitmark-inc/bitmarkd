// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package announce

// add a certificate
func AddCertificate(fingerprint [32]byte, certificate []byte) {
	//announce.certificatePool.Add(fingerprint[:], certificate)
}

// // fetch a certificate
// func GetCertificate(fingerprint *util.FingerprintBytes) []byte {
// 	return announce.certificatePool.Get(fingerprint[:])
// }

// // certificate already stored
// func HasCertificate(fingerprint *util.FingerprintBytes) bool {
// 	return announce.certificatePool.Has(fingerprint[:])
// }
