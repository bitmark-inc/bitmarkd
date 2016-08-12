// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package block

import (
	"encoding/binary"
	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"sync"
)

// from storage/doc.go:
//
//   owner                 - next count value to use for appending to owned items
//                           data: count
//   owner ++ count        - list of owned items
//                           data: txid ++ issue txid ++ asset index
//   owner ++ txid         - position in list of owned items, for delete after transfer
//                           data: count

var toLock sync.Mutex

// need to have a lock
func TransferOwnership(link merkle.Digest, currentOwner *account.Account, newOwner *account.Account) {

	// ensure single threaded
	toLock.Lock()
	defer toLock.Unlock()

	// get count for current owner record
	dKey := append(currentOwner.Bytes(), link[:]...)
	dCount := storage.Pool.OwnerDigest.Get(dKey)
	if nil == dCount {
		fault.Panic("TransferOwnership: OwnerDigest database corrupt")
	}

	// delete the current owners records
	oKey := append(currentOwner.Bytes(), dCount...)
	previousData := storage.Pool.Ownership.Get(oKey)
	if nil == previousData {
		fault.Panic("TransferOwnership: Ownership database corrupt")
	}
	storage.Pool.Ownership.Delete(oKey)
	storage.Pool.OwnerDigest.Delete(dKey)

	// if no new owner only above delete was needed
	if nil == newOwner {
		return
	}

	// create the owner data
	newData := append(link[:], previousData[merkle.DigestLength:]...)
	create(link, newData, newOwner)
}

// internal creation routine, must be called with lock held
func create(link merkle.Digest, ownerData []byte, owner *account.Account) {

	// increment the count for new owner
	nKey := owner.Bytes()
	count := storage.Pool.OwnerCount.Get(nKey)
	if nil == count {
		count = []byte{0, 0, 0, 0, 0, 0, 0, 0}
	} else if 8 != len(count) {
		fault.Panic("CreateOwnership: OwnerCount database corrupt")
	}
	newCount := make([]byte, 8)
	binary.BigEndian.PutUint64(newCount, binary.BigEndian.Uint64(count)+1)
	storage.Pool.OwnerCount.Put(nKey, newCount)

	// write the new owner
	oKey := append(owner.Bytes(), count...)

	// txid++ issue id + asset index
	storage.Pool.Ownership.Put(oKey, ownerData)

	// write new digest record
	dKey := append(owner.Bytes(), link[:]...)
	storage.Pool.OwnerDigest.Put(dKey, count)
}

func CreateOwnership(link merkle.Digest, assetIndex transactionrecord.AssetIndex, blockNumber uint64, newOwner *account.Account) {
	// ensure single threaded
	toLock.Lock()
	defer toLock.Unlock()

	// 8 byte block number
	blk := make([]byte, 8)
	binary.BigEndian.PutUint64(blk, blockNumber)

	// create a new owner data value
	newData := append(link[:], link[:]...)
	newData = append(newData, assetIndex[:]...)
	newData = append(newData, blk...)

	// store to database
	create(link, newData, newOwner)
}

// find the owner of a specific transaction
// (only issue or transfer is allowed)
func OwnerOf(link merkle.Digest) *account.Account {

	key := link[:]
	packed := storage.Pool.Transactions.Get(key)
	if nil == packed {
		return nil
	}

	transaction, _, err := transactionrecord.Packed(packed).Unpack()
	fault.PanicIfError("block.OwnerOf", err)

	switch transaction.(type) {
	case *transactionrecord.BitmarkIssue:
		issue := transaction.(*transactionrecord.BitmarkIssue)
		return issue.Owner

	case *transactionrecord.BitmarkTransfer:
		transfer := transaction.(*transactionrecord.BitmarkTransfer)
		return transfer.Owner

	default:
		fault.Panicf("block.OwnerOf: incorrect transaction: %v", transaction)
		return nil
	}
}
