// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package block

import (
	"encoding/binary"
	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
	"sync"
)

// from storage/doc.go:
//
//   owner                 - next count value to use for appending to owned items
//                           data: count
//   owner ++ count        - list of owned items
//                           data: last transfer txId ++ last transfer block number ++ issue txId ++ issue block number ++ asset index
//   owner ++ txId         - position in list of owned items, for delete after transfer
//                           data: count

// to ensure synchronised ownership updates
var toLock sync.Mutex

// structure of the ownership record
const (
	txIdStart  = 0
	txIdFinish = txIdStart + merkle.DigestLength

	transferBlockNumberStart  = txIdFinish
	transferBlockNumberFinish = transferBlockNumberStart + 8

	remainderStart = transferBlockNumberFinish // everything after transfer data

	issueTxIdStart  = transferBlockNumberFinish
	issueTxIdFinish = issueTxIdStart + merkle.DigestLength

	issueBlockNumberStart  = issueTxIdFinish
	issueBlockNumberFinish = issueBlockNumberStart + 8

	assetIndexStart  = issueBlockNumberFinish
	assetIndexFinish = assetIndexStart + transactionrecord.AssetIndexLength
)

// need to have a lock
func TransferOwnership(previousTxId merkle.Digest, transferTxId merkle.Digest, transferBlockNumber uint64, currentOwner *account.Account, newOwner *account.Account) {

	// ensure single threaded
	toLock.Lock()
	defer toLock.Unlock()

	// get count for current owner record
	dKey := append(currentOwner.Bytes(), previousTxId[:]...)
	dCount := storage.Pool.OwnerDigest.Get(dKey)
	if nil == dCount {
		logger.Criticalf("TransferOwnership: dKey: %x", dKey)
		logger.Criticalf("TransferOwnership: block number: %d", transferBlockNumber)
		logger.Criticalf("TransferOwnership: previous tx id: %#v", previousTxId)
		logger.Criticalf("TransferOwnership: transfer tx id: %#v", transferTxId)
		logger.Criticalf("TransferOwnership: current owner: %x  %v", currentOwner.Bytes(), currentOwner)
		logger.Criticalf("TransferOwnership: new     owner: %x  %v", newOwner.Bytes(), newOwner)

		// ow, err := ListBitmarksFor(currentOwner, 0, 999)
		// if nil != err {
		// 	logger.Criticalf("lbf: error: %s", err)
		// } else {
		// 	logger.Criticalf("lbf: %#v", ow)
		// }

		logger.Panic("TransferOwnership: OwnerDigest database corrupt")
	}

	// delete the current owners records
	oKey := append(currentOwner.Bytes(), dCount...)
	ownerData := storage.Pool.Ownership.Get(oKey)
	if nil == ownerData {
		logger.Panic("TransferOwnership: Ownership database corrupt")
	}
	storage.Pool.Ownership.Delete(oKey)
	storage.Pool.OwnerDigest.Delete(dKey)

	// if no new owner only above delete was needed
	if nil == newOwner {
		return
	}

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
		logger.Panic("CreateOwnership: OwnerCount database corrupt")
	}
	newCount := make([]byte, 8)
	binary.BigEndian.PutUint64(newCount, binary.BigEndian.Uint64(count)+1)
	storage.Pool.OwnerCount.Put(nKey, newCount)

	// write the new owner
	oKey := append(owner.Bytes(), count...)

	// txId ++ last transfer block number ++ issue txId ++ issue block number ++ asset index
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

	// create a new owner data value:
	//   Issue id ++ zero  block number  -- replaced by sucessive: transfer id ++ transfer block number
	//   Issue id ++ issue block number  -- will remain constant
	//   asset index
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
	logger.PanicIfError("block.OwnerOf", err)

	switch tx := transaction.(type) {
	case *transactionrecord.BitmarkIssue:
		return tx.Owner

	case *transactionrecord.BitmarkTransferUnratified:
		return tx.Owner

	case *transactionrecord.BitmarkTransferCountersigned:
		return tx.Owner

	default:
		logger.Panicf("block.OwnerOf: incorrect transaction: %v", transaction)
		return nil
	}
}

// type to represent an ownership record
type Ownership struct {
	N          uint64                       `json:"n,string"`
	TxId       merkle.Digest                `json:"txId"`
	IssueTxId  merkle.Digest                `json:"issue"`
	AssetIndex transactionrecord.AssetIndex `json:"index"`
}

// fetch a list of bitmarks for an owner
func ListBitmarksFor(owner *account.Account, start uint64, count int) ([]Ownership, error) {

	startBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(startBytes, start)
	prefix := append(owner.Bytes(), startBytes...)

	cursor := storage.Pool.Ownership.NewFetchCursor().Seek(prefix)

	items, err := cursor.Fetch(count)
	if nil != err {
		return nil, err
	}

	records := make([]Ownership, len(items))

	for i, item := range items {
		n := len(item.Key)
		records[i].N = binary.BigEndian.Uint64(item.Key[n-8:])
		merkle.DigestFromBytes(&records[i].TxId, item.Value[txIdStart:txIdFinish])
		merkle.DigestFromBytes(&records[i].IssueTxId, item.Value[issueTxIdStart:issueTxIdFinish])
		transactionrecord.AssetIndexFromBytes(&records[i].AssetIndex, item.Value[assetIndexStart:assetIndexFinish])
	}

	return records, nil
}
