// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package transaction

import (
	"github.com/bitmark-inc/bitmarkd/fault"
)

// fetch some transactions for client
func FetchPending() []Decoded {

	startIndex := []byte{}
	txids := make([]Link, 0, 100)

loop:
	for {
		// read blocks of records
		state, err := transactionPool.statePool.Fetch(startIndex, 100)
		if nil != err {
			// error represents a database failure - panic
			fault.Criticalf("transaction.FetchPending: statePool.Fetch failed, err = %v", err)
			fault.Panic("transaction.FetchPending: failed")
		}

		// if only one or zero records exit loop
		n := len(state)
		if n <= 1 {
			break loop
		}

		// last key for next loop
		startIndex = state[n-1].Key

		// exclude the mined transactions
		for _, e := range state {
			if MinedTransaction == State(e.Value[0]) {
				continue
			}
			var txid Link
			LinkFromBytes(&txid, e.Key) // the transaction id
			txids = append(txids, txid)
		}
	}

	return Decode(txids)
}
