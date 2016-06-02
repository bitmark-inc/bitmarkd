// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package block

import (
	"fmt"
	"io"
)

func PrintBlockTimes(fh io.Writer, beginBlockNumber uint64, endBlockNumber uint64) {

	n := Number()

	initialised := false
	lastTimestampSeconds := int64(0)

	if 0 == beginBlockNumber || beginBlockNumber <= GenesisBlockNumber {
		beginBlockNumber = GenesisBlockNumber
	}
	if 0 == endBlockNumber || endBlockNumber >= n-1 {
		endBlockNumber = n - 1
	}

	fmt.Fprintf(fh, "%q %q %q %q %q\n", "block number", "timestamp", "minutes", "pool difficulty", "Tx Count")
	for blockNumber := beginBlockNumber; blockNumber <= endBlockNumber; blockNumber += 1 {

		packed := Get(blockNumber)
		if nil == packed {
			fmt.Fprintf(fh, "%d ***MISSING***\n", blockNumber)
			continue
		}

		var blk Block
		err := packed.Unpack(&blk)
		if nil != err {
			fmt.Fprintf(fh, "%d ***ERROR: %v ***\n", blockNumber, err)
			continue
		}

		timestampSeconds := blk.Timestamp.Unix()
		if !initialised {
			lastTimestampSeconds = timestampSeconds
			initialised = true
		}
		delta := timestampSeconds - lastTimestampSeconds
		lastTimestampSeconds = timestampSeconds
		deltaMinutes := float64(delta) / 60

		txIdCount := len(blk.TxIds)
		pdiff := blk.Header.Bits.Pdiff()

		fmt.Fprintf(fh, "%d %d %f %f %d\n", blockNumber, timestampSeconds, deltaMinutes, pdiff, txIdCount)
	}
}
