// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package block_test

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/bitmark-inc/bitmarkd/block"
	"testing"
	"time"
)

// create a split coinbase transaction for the miner
func TestNewCoinbase(t *testing.T) {

	blockNumbers := []struct {
		n uint64
		b string
	}{
		{0, "0000"},
		{1, "0100"},
		{2, "0200"},
		{3, "0300"},
		{127, "7f00"},
		{128, "8000"},
		{129, "8100"},
		{255, "ff00"},
		{256, "0001"},
		{257, "0101"},
		{65535, "ffff"},
		{65536, "000001"},
		{65537, "010001"},
		{1048575, "ffff0f"},
		{1048576, "000010"},
		{0xffffffffff, "ffffffffff"},
	}

	for _, bl := range blockNumbers {

		blockNumber := bl.n
		blockNumberHex := bl.b
		blockNumberLength := len(blockNumberHex) / 2
		blockNumberLengthHex := fmt.Sprintf("%02x", blockNumberLength)

		timestamp := time.Now().UTC()
		timestampLength := 4
		timestampBuffer := make([]byte, 8)
		binary.LittleEndian.PutUint64(timestampBuffer, uint64(timestamp.Unix()))

		for i := 8; i > 4; i -= 1 {
			if 0 != timestampBuffer[i-1] {
				timestampLength = i
				break
			}
		}
		timestampHex := fmt.Sprintf("%x", timestampBuffer[:timestampLength])
		timestampLengthHex := fmt.Sprintf("%02x", timestampLength)

		nonceSize := 8
		nonceSizeHex := fmt.Sprintf("%02x", nonceSize)

		opcodeCount := 3
		scriptLength := opcodeCount + blockNumberLength + timestampLength + nonceSize
		scriptLengthHex := fmt.Sprintf("%02x", scriptLength)

		// for CB2

		mAddress := block.MinerAddress{
			Currency: "justtesting",
			Address:  "this-is-a-test",
		}
		address := mAddress.String()
		addressHex := hex.EncodeToString([]byte(address))

		outScriptLength := 1 + len(address)
		outScriptLengthHex := fmt.Sprintf("%02x", outScriptLength)

		// run coinbase create
		binaryCB1, binaryCB2 := block.NewCoinbase(blockNumber, timestamp, nonceSize, []block.MinerAddress{mAddress})
		cb1 := hex.EncodeToString(binaryCB1)
		cb2 := hex.EncodeToString(binaryCB2)

		expectedCB1 := "01000000010000000000000000000000000000000000000000000000000000000000000000ffffffff" +
			scriptLengthHex +
			blockNumberLengthHex + blockNumberHex +
			timestampLengthHex + timestampHex +
			nonceSizeHex
		expectedCB2 := "00000000" + // remainder of in
			"01" + // outs count
			"0000000000000000" + outScriptLengthHex + "6a" + addressHex + "00000000" // out[0]

		if cb1 != expectedCB1 {
			t.Errorf("actual cb1: %s", cb1)
			t.Errorf("  expected: %s", expectedCB1)
		}
		if cb2 != expectedCB2 {
			t.Errorf("actual cb2: %s", cb2)
			t.Errorf("  expected: %s", expectedCB2)
		}
	}
}
