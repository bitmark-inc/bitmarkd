// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package block_test

import (
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/genesis"
	"testing"
)

func TestBitmarkCRC(t *testing.T) {

	// dependant on the genesis digest for bitmark
	expected := uint64(0xd73c9f4000217194)

	actual := block.CRC(genesis.BlockNumber, genesis.LiveGenesisBlock)
	if expected != actual {
		t.Fatalf("crc expected: 0x%016x  actual: 0x%016x", expected, actual)
	}
}

func TestTestingCRC(t *testing.T) {

	// dependant on the genesis digest for testing
	expected := uint64(0xd1cc53a056227402)

	actual := block.CRC(genesis.BlockNumber, genesis.TestGenesisBlock)
	if expected != actual {
		t.Fatalf("crc expected: 0x%016x  actual: 0x%016x", expected, actual)
	}
}
