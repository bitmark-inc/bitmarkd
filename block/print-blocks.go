// Copyright (c) 2014-2015 Bitmark Inc.
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
	lastSeconds := int64(0)

	if 0 == beginBlockNumber || beginBlockNumber <= GenesisBlockNumber {
		beginBlockNumber = GenesisBlockNumber
	}
	if 0 == endBlockNumber || endBlockNumber >= n - 1 {
		endBlockNumber = n - 1
	}

	fmt.Fprintf(fh, "%q %q %q %q\n", "block number", "timestamp", "minutes", "pool difficulty")
	for blockNumber := beginBlockNumber; blockNumber <= endBlockNumber; blockNumber += 1 {

		packed, exists := Get(blockNumber)
		if !exists {
			fmt.Fprintf(fh, "%d ***MISSING***\n", blockNumber)
			continue
		}

		var blk Block
		err := packed.Unpack(&blk)
		if nil != err {
			fmt.Fprintf(fh, "%d ***ERROR: %v ***\n", blockNumber, err)
			continue
		}

		seconds := blk.Timestamp.Unix()
		if !initialised {
			lastSeconds = seconds
			initialised = true
		}
		delta := seconds - lastSeconds
		lastSeconds = seconds

		fmt.Fprintf(fh, "%d %d %f %f\n", blockNumber, seconds, float64(delta)/60, blk.Header.Bits.Pdiff())
	}
}
