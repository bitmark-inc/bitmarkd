// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package transaction

import (
	"encoding/binary"
	"github.com/bitmark-inc/bitmarkd/fault"
	"time"
)

// type to hold unpaid items
type UnpaidResult struct {
	Link      Link
	Timestamp time.Time
}

// fetch some transaction ids for payment verification
//
// returns:
//   list of ids
//   next start point
func (cursor *IndexCursor) FetchUnpaid(count int) []UnpaidResult {

	unpaid, err := transactionPool.pendingPool.Fetch(cursor.Bytes(), count)
	if nil != err {
		// error represents a database failure - panic
		fault.Criticalf("transaction.FetchUnpaid: unpaidPool.Fetch failed, err = %v", err)
		fault.Panic("transaction.FetchUnpaid: failed")
	}

	length := len(unpaid)

	// if nothing unpaid just return the same cursor value
	if 0 == length {
		return nil
	}

	results := make([]UnpaidResult, length)

	for i, e := range unpaid {
		LinkFromBytes(&results[i].Link, e.Value[:LinkSize]) // the transaction id
		if len(e.Value) > LinkSize {
			seconds := binary.BigEndian.Uint64(e.Value[LinkSize:]) // the creation time
			results[i].Timestamp = time.Unix(int64(seconds), 0).UTC()
		} else {
			results[i].Timestamp = time.Unix(0, 0).UTC()
		}
		// update cursor
		*cursor = IndexCursor(binary.BigEndian.Uint64(e.Key) + 1)

	}

	return results
}
