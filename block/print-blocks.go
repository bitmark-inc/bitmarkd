// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package block

import (
	"fmt"
	"io"
)

func PrintBlockTimes(fh io.Writer) {

	n := Number()
	fmt.Fprintf(fh, "%q %q %q %q\n", "block number", "timestamp", "minutes", "pool difficulty")

	initialised := false
	lastSeconds := int64(0)

	// note: does not output the genesis block
	for blockNumber := GenesisBlockNumber + 1; blockNumber < n; blockNumber += 1 {

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
