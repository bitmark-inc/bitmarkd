// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package ownership

import (
	"encoding/binary"
	"sync"

	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
)

// from storage/doc.go:
//
//   N ++ owner            - next count value to use for appending to owned items
//                           data: count
//   K ++ owner ++ count   - list of owned items
//                           data: 00 ++ last transfer txId ++ last transfer BN ++ issue txId ++ issue BN ++ asset id
//                           data: 01 ++ last transfer txId ++ last transfer BN ++ issue txId ++ issue BN ++ owned BN
//   D ++ owner ++ txId    - position in list of owned items, for delete after transfer

// to ensure synchronised ownership updates
var toLock sync.Mutex

const (
	oneByteSize    = 1
	uint64ByteSize = 8
)

// structure of the ownership record
const (
	FlagByteStart  = 0
	FlagByteFinish = FlagByteStart + oneByteSize

	TxIdStart  = FlagByteFinish
	TxIdFinish = TxIdStart + merkle.DigestLength

	TransferBlockNumberStart  = TxIdFinish
	TransferBlockNumberFinish = TransferBlockNumberStart + uint64ByteSize

	IssueTxIdStart  = TransferBlockNumberFinish
	IssueTxIdFinish = IssueTxIdStart + merkle.DigestLength

	IssueBlockNumberStart  = IssueTxIdFinish
	IssueBlockNumberFinish = IssueBlockNumberStart + uint64ByteSize

	// overlap flag==0x00
	AssetIdentifierStart  = IssueBlockNumberFinish
	AssetIdentifierFinish = AssetIdentifierStart + transactionrecord.AssetIdentifierLength

	// overlap flag==0x01
	OwnedBlockNumberStart  = IssueBlockNumberFinish
	OwnedBlockNumberFinish = OwnedBlockNumberStart + uint64ByteSize
)

// need to have a lock
func Transfer(previousTxId merkle.Digest, transferTxId merkle.Digest, transferBlockNumber uint64, currentOwner *account.Account, newOwner *account.Account) {

	// ensure single threaded
	toLock.Lock()
	defer toLock.Unlock()

	// get count for current owner record
	dKey := append(currentOwner.Bytes(), previousTxId[:]...)
	dCount := storage.Pool.OwnerDigest.Get(dKey)
	if nil == dCount {
		logger.Criticalf("ownership.Transfer: dKey: %x", dKey)
		logger.Criticalf("ownership.Transfer: block number: %d", transferBlockNumber)
		logger.Criticalf("ownership.Transfer: previous tx id: %#v", previousTxId)
		logger.Criticalf("ownership.Transfer: transfer tx id: %#v", transferTxId)
		logger.Criticalf("ownership.Transfer: current owner: %x  %v", currentOwner.Bytes(), currentOwner)
		if nil != newOwner {
			logger.Criticalf("ownership.Transfer: new     owner: %x  %v", newOwner.Bytes(), newOwner)
		}

		// ow, err := ListBitmarksFor(currentOwner, 0, 999)
		// if nil != err {
		// 	logger.Criticalf("lbf: error: %s", err)
		// } else {
		// 	logger.Criticalf("lbf: %#v", ow)
		// }

		logger.Panic("ownership.Transfer: OwnerDigest database corrupt")
	}

	// delete the current owners records
	oKey := append(currentOwner.Bytes(), dCount...)
	ownerData := storage.Pool.Ownership.Get(oKey)
	if nil == ownerData {
		logger.Criticalf("ownership.Transfer: no ownerData for key: %x", oKey)
		logger.Panic("ownership.Transfer: Ownership database corrupt")
	}
	storage.Pool.Ownership.Delete(oKey)
	storage.Pool.OwnerDigest.Delete(dKey)

	// if no new owner only above delete was needed
	if nil == newOwner {
		return
	}

	copy(ownerData[TxIdStart:TxIdFinish], transferTxId[:])
	binary.BigEndian.PutUint64(ownerData[TransferBlockNumberStart:TransferBlockNumberFinish], transferBlockNumber)
	create(transferTxId, ownerData, newOwner)
}

// internal creation routine, must be called with lock held
func create(txId merkle.Digest, ownerData []byte, owner *account.Account) {

	// increment the count for new owner
	nKey := owner.Bytes()
	count := storage.Pool.OwnerCount.Get(nKey)
	if nil == count {
		count = []byte{0, 0, 0, 0, 0, 0, 0, 0}
	} else if uint64ByteSize != len(count) {
		logger.Panic("CreateOwnership: OwnerCount database corrupt")
	}
	newCount := make([]byte, uint64ByteSize)
	binary.BigEndian.PutUint64(newCount, binary.BigEndian.Uint64(count)+1)
	storage.Pool.OwnerCount.Put(nKey, newCount)

	// write the new owner
	oKey := append(owner.Bytes(), count...)

	// flag ++ txId ++ last transfer block number ++ issue txId ++ issue block number ++ AssetIdentifier/BlockNumber
	storage.Pool.Ownership.Put(oKey, ownerData)

	// write new digest record
	dKey := append(owner.Bytes(), txId[:]...)
	storage.Pool.OwnerDigest.Put(dKey, count)
}

func CreateAsset(issueTxId merkle.Digest, issueBlockNumber uint64, assetId transactionrecord.AssetIdentifier, newOwner *account.Account) {
	// ensure single threaded
	toLock.Lock()
	defer toLock.Unlock()

	// 8 byte block number
	blk := make([]byte, uint64ByteSize)
	binary.BigEndian.PutUint64(blk, issueBlockNumber)

	// create a new owner data value:
	//   flag = OwnedAsset
	//   issue id ++ zero  block number  -- replaced by sucessive: transfer id ++ transfer block number
	//   issue id ++ issue block number  -- will remain constant
	//   asset id
	newData := append([]byte{byte(OwnedAsset)}, issueTxId[:]...)
	newData = append(newData, []byte{0, 0, 0, 0, 0, 0, 0, 0}...)
	newData = append(newData, issueTxId[:]...)
	newData = append(newData, blk...)
	newData = append(newData, assetId[:]...)

	// store to database
	create(issueTxId, newData, newOwner)
}

func CreateBlock(issueTxId merkle.Digest, blockNumber uint64, newOwner *account.Account) {
	// ensure single threaded
	toLock.Lock()
	defer toLock.Unlock()

	// 8 byte block number
	blk := make([]byte, uint64ByteSize)
	binary.BigEndian.PutUint64(blk, blockNumber)

	// create a new owner data value:
	//   flag = OwnedBlock
	//   issue id ++ zero  block number  -- replaced by sucessive: transfer id ++ transfer block number
	//   issue id ++ issue block number  -- will remain constant
	//   block number
	newData := append([]byte{byte(OwnedBlock)}, issueTxId[:]...)
	newData = append(newData, []byte{0, 0, 0, 0, 0, 0, 0, 0}...)
	newData = append(newData, issueTxId[:]...)
	newData = append(newData, blk...)
	newData = append(newData, blk...)

	// store to database
	create(issueTxId, newData, newOwner)
}

// find the owner of a specific transaction
func OwnerOf(txId merkle.Digest) *account.Account {

	_, packed := storage.Pool.Transactions.GetNB(txId[:]) // drop block number
	if nil == packed {
		return nil
	}

	transaction, _, err := transactionrecord.Packed(packed).Unpack(mode.IsTesting())
	logger.PanicIfError("ownership.OwnerOf", err)

	switch tx := transaction.(type) {
	case *transactionrecord.BitmarkIssue:
		return tx.Owner

	case *transactionrecord.BitmarkTransferUnratified:
		return tx.Owner

	case *transactionrecord.BitmarkTransferCountersigned:
		return tx.Owner

	case *transactionrecord.BlockFoundation:
		return tx.Owner

	case *transactionrecord.BlockOwnerTransfer:
		return tx.Owner

	default:
		logger.Panicf("block.OwnerOf: incorrect transaction: %v", transaction)
		return nil
	}
}

// find owner currently owns this transaction id
func CurrentlyOwns(owner *account.Account, txId merkle.Digest) bool {
	dKey := append(owner.Bytes(), txId[:]...)
	return nil != storage.Pool.OwnerDigest.Get(dKey)
}
