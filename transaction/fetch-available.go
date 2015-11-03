// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package transaction

import (
	"encoding/binary"
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/fault"
)

// fetch a list of transaction ids for miner
//
// returns:
//   list of ids (as digests for merkle tree processing later)
//
// note that an asset txId can be inserted just before an issue txId
//      if that asset was never seen before
func (cursor *AvailableCursor) FetchAvailable(count int) []block.Digest {

	fetchCursor := transactionPool.verifiedPool.NewFetchCursor()
	fetchCursor.Seek(cursor.count.Bytes())

	verified, err := fetchCursor.Fetch(count)
	if nil != err {
		// error represents a database failure - panic
		fault.PanicWithError("transaction.FetchAvailable: verifiedPool.Fetch", err)
	}

	length := len(verified)

	// if nothing verified just return the same cursor value
	if 0 == length {
		return nil
	}

	results := make([]block.Digest, 0, count)

loop:
	for _, e := range verified {

		var txId Link
		LinkFromBytes(&txId, e.Value[:LinkSize])

		state, packedTx, found := txId.Read()
		if !found || VerifiedTransaction != state {
			// error represents a database failure - panic
			fault.Criticalf("transaction.FetchAvailable: problem TxId: %#v  state: %s found: %v", txId, state, found)
			//fault.Panic("transaction.FetchAvailable: read tx problem")

			// if the tx disappeared then skip it (maybe was mined)
			continue loop
		}

		unpackedTx, err := packedTx.Unpack()
		if nil != err {
			fault.PanicWithError("transaction.FetchAvailable: unpack", err)
		}

		// check if an issue
		switch unpackedTx.(type) {
		case *BitmarkIssue:
			issue := unpackedTx.(*BitmarkIssue)
			state, link, found := issue.AssetIndex.Read()
			if !found {
				fault.Criticalf("transaction.FetchAvailable: TxId: %#v problem with asset index: %#v", txId, issue.AssetIndex)
				continue // skip any issues lacking asset
			}
			if ConfirmedTransaction != state {
				if _, ok := cursor.assets[link]; !ok {

					results = append(results, block.Digest(link))

					// flag asset in map to prevent duplicates
					cursor.assets[link] = struct{}{}

					// if adding asset causes count to reached
					if len(results) >= count {
						// set cursor so that this issue will be first next time
						cursor.count = IndexCursor(binary.BigEndian.Uint64(e.Key))
						break loop
					}
				}
			}

		default:
		}

		results = append(results, block.Digest(txId))
		// update cursor so that next record will be output next time
		cursor.count = IndexCursor(binary.BigEndian.Uint64(e.Key) + 1)
	}

	return results
}
