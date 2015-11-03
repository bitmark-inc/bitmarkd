// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package transaction

import (
	"github.com/bitmark-inc/bitmarkd/fault"
)

// fetch some transactions for client
func FetchPending() []Decoded {

	stateCursor := transactionPool.statePool.NewFetchCursor()
	txids := make([]Link, 0, 100)

loop:
	for {
		// read blocks of records
		state, err := stateCursor.Fetch(100)
		if nil != err {
			// error represents a database failure - panic
			fault.Criticalf("transaction.FetchPending: statePool.Fetch failed, err = %v", err)
			fault.Panic("transaction.FetchPending: failed")
		}

		// if only one or zero records exit loop
		if 0 == len(state) {
			break loop
		}

		// exclude the mined transactions
		for _, e := range state {
			if ConfirmedTransaction == State(e.Value[0]) {
				continue
			}
			var txid Link
			LinkFromBytes(&txid, e.Key) // the transaction id
			txids = append(txids, txid)
		}
	}

	return Decode(txids)
}
