// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockring_test

import (
	"testing"

	"github.com/bitmark-inc/bitmarkd/blockring"
	"github.com/bitmark-inc/bitmarkd/genesis"
)

func TestBitmarkCRC(t *testing.T) {

	// dependant on the genesis digest for bitmark
	expected := uint64(0x445f81247a6fdecc)

	actual := blockring.CRC(genesis.BlockNumber, genesis.LiveGenesisBlock)
	if expected != actual {
		t.Fatalf("crc expected: 0x%016x  actual: 0x%016x", expected, actual)
	}
}

func TestTestingCRC(t *testing.T) {

	// dependant on the genesis digest for testing
	expected := uint64(0xd1cc53a056227402)

	actual := blockring.CRC(genesis.BlockNumber, genesis.TestGenesisBlock)
	if expected != actual {
		t.Fatalf("crc expected: 0x%016x  actual: 0x%016x", expected, actual)
	}
}
