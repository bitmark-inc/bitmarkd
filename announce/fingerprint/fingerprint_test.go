// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package fingerprint_test

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/bitmarkd/announce/fingerprint"
)

func TestMarshalText(t *testing.T) {
	f := fingerprint.Type{0, 1, 2, 3, 4, 5}

	size := hex.EncodedLen(len(f))
	buffer := make([]byte, size)
	hex.Encode(buffer, f[:])

	marshaled, err := f.MarshalText()
	assert.Nil(t, err, "wrong error")
	assert.Equal(t, buffer, marshaled, "wrong content")
}
