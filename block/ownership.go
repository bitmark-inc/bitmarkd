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
//                           data: last transfer txid ++ last transfer block number ++ issue txid ++ asset index ++ issue block number
//   owner ++ txid         - position in list of owned items, for delete after transfer
//                           data: count

var toLock sync.Mutex

// need to have a lock
func TransferOwnership(previousTxId merkle.Digest, transferTxId merkle.Digest, transferBlockNumber uint64, currentOwner *account.Account, newOwner *account.Account) {

	// ensure single threaded
	toLock.Lock()
	defer toLock.Unlock()

	// get count for current owner record
	dKey := append(currentOwner.Bytes(), previousTxId[:]...)
	dCount := storage.Pool.OwnerDigest.Get(dKey)
	if nil == dCount {
		fault.Panic("TransferOwnership: OwnerDigest database corrupt")
	}

	// delete the current owners records
	oKey := append(currentOwner.Bytes(), dCount...)
	ownerData := storage.Pool.Ownership.Get(oKey)
	if nil == ownerData {
		fault.Panic("TransferOwnership: Ownership database corrupt")
	}
	storage.Pool.Ownership.Delete(oKey)
	storage.Pool.OwnerDigest.Delete(dKey)

	// if no new owner only above delete was needed
	if nil == newOwner {
		return
	}

	// create the owner data by replacing txId and its block number
	const (
		txIdStart                 = 0
		txIdFinish                = merkle.DigestLength
		transferBlockNumberStart  = txIdFinish
		transferBlockNumberFinish = transferBlockNumberStart + 8
		remainderStart            = transferBlockNumberFinish
	)

	copy(ownerData[txIdStart:txIdFinish], transferTxId[:])
	binary.BigEndian.PutUint64(ownerData[transferBlockNumberStart:transferBlockNumberFinish], transferBlockNumber)
	create(transferTxId, ownerData, newOwner)
}

// internal creation routine, must be called with lock held
func create(txId merkle.Digest, ownerData []byte, owner *account.Account) {

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
	dKey := append(owner.Bytes(), txId[:]...)
	storage.Pool.OwnerDigest.Put(dKey, count)
}

func CreateOwnership(issueTxId merkle.Digest, issueBlockNumber uint64, assetIndex transactionrecord.AssetIndex, newOwner *account.Account) {
	// ensure single threaded
	toLock.Lock()
	defer toLock.Unlock()

	// 8 byte block number
	blk := make([]byte, 8)
	binary.BigEndian.PutUint64(blk, issueBlockNumber)

	// create a new owner data value
	newData := append(issueTxId[:], []byte{0, 0, 0, 0, 0, 0, 0, 0}...)
	newData = append(newData, issueTxId[:]...)
	newData = append(newData, blk...)
	newData = append(newData, assetIndex[:]...)

	// store to database
	create(issueTxId, newData, newOwner)
}

// find the owner of a specific transaction
// (only issue or transfer is allowed)
func OwnerOf(txId merkle.Digest) *account.Account {

	key := txId[:]
	packed := storage.Pool.Transactions.Get(key)
	if nil == packed {
		return nil
	}

	transaction, _, err := transactionrecord.Packed(packed).Unpack()
	fault.PanicIfError("block.OwnerOf", err)

	switch tx := transaction.(type) {
	case *transactionrecord.BitmarkIssue:
		return tx.Owner

	case *transactionrecord.BitmarkTransfer:
		return tx.Owner

	default:
		fault.Panicf("block.OwnerOf: incorrect transaction: %v", transaction)
		return nil
	}
}