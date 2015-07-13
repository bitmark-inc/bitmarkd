// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package transaction

import (
	"encoding/binary"
	"github.com/bitmark-inc/bitmarkd/pool"
	"time"
)

// type to hold pool items
type PoolResult struct {
	Transaction Decoded
	Timestamp   time.Time
}

// fetch some transaction ids from pool
//
// returns:
//   list of ids
//   cursor set to next start point
func (cursor *IndexCursor) FetchPool(count int) []PoolResult {
	// pick a default count
	if count <= 0 {
		count = 10
	}

	transactionPool.Lock()
	defer transactionPool.Unlock()

	itPending := transactionPool.pendingPool.Iterate(cursor.Bytes())
	defer itPending.Release()

	itVerified := transactionPool.verifiedPool.Iterate(cursor.Bytes())
	defer itVerified.Release()

	ePending := (*pool.Element)(nil)
	eVerified := (*pool.Element)(nil)

	txIds := make([]Link, 1)

	results := make([]PoolResult, count)
	nextIndex := *cursor
	length := 0
	for n := 0; n < count; n += 1 {

		if nil == ePending {
			ePending = itPending.Next()
		}
		if nil == eVerified {
			eVerified = itVerified.Next()
		}

		if nil == ePending && nil == eVerified {
			break
		}

		e := (*pool.Element)(nil)

		if nil != ePending && nil != eVerified {
			kp := binary.BigEndian.Uint64(ePending.Key)
			kv := binary.BigEndian.Uint64(eVerified.Key)

			if kp < kv {
				e = ePending
				ePending = nil
			} else {
				e = eVerified
				eVerified = nil
			}
		} else if nil != ePending {
			e = ePending
			ePending = nil
		} else if nil != eVerified {
			e = eVerified
			eVerified = nil
		}

		nextIndex = IndexCursor(binary.BigEndian.Uint64(e.Key) + 1)

		LinkFromBytes(&txIds[0], e.Value[:LinkSize]) // the transaction id
		t := Decode(txIds)
		results[n].Transaction = t[0]

		seconds := binary.BigEndian.Uint64(e.Value[LinkSize:]) // the creation time
		results[n].Timestamp = time.Unix(int64(seconds), 0).UTC()
		length += 1
	}

	*cursor = nextIndex
	return results[:length]
}
