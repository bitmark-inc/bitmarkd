// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package fingerprint

import "encoding/hex"

// type for SHA3 fingerprints
type Fingerprint [32]byte

// MarshalText - convert fingerprint to little endian hex text
func (fingerprint Fingerprint) MarshalText() ([]byte, error) {
	size := hex.EncodedLen(len(fingerprint))
	buffer := make([]byte, size)
	hex.Encode(buffer, fingerprint[:])
	return buffer, nil
}
