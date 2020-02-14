// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package proof_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/bitmarkd/counter"
	"github.com/bitmark-inc/bitmarkd/proof"
)

func TestMinedBlocks(t *testing.T) {
	assert.Equal(t, counter.Counter(0), proof.MinedBlocks(), "wrong init value")
}

func TestFailMinedBlocks(t *testing.T) {
	assert.Equal(t, counter.Counter(0), proof.FailMinedBlocks(), "wrong init value")
}
