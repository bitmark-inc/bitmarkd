// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrivateKeyDecodeEncode(t *testing.T) {
	originKey := "080112406eb84a3845d33c2a389d7fbea425cbf882047a2ab13084562f06875db47b5fdc2e45a298e6cd0472eeb97cd023c723824e157869d81039794864987c05b212a8"
	k, err := DecodePrivKeyFromHex(originKey)
	assert.NoError(t, err, "Decode Hex Key Error")

	revertKey, err := EncodePrivKeyToHex(k)
	assert.NoError(t, err, "Decode Hex Key Error")
	assert.Equal(t, originKey, revertKey)
}
