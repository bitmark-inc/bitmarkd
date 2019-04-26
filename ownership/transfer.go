// Copyright (c) 2014-2019 Bitmark Inc.
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

// to ensure synchronised ownership updates
var toLock sync.Mutex

// from storage/setup.go:
//
// Ownership:
//   OwnerNextCount  BN   - next count value to use for appending to owned items
//   OwnerList       txId - list of owned items
//   OwnerTxIndex    BN   - position in list of owned items, for delete after transfer

// Share - setup for share ownership transfer, must have a lock
func Share(previousTxId merkle.Digest, transferTxId merkle.Digest, transferBlockNumber uint64, currentOwner *account.Account, balance uint64) {

	// ensure single threaded
	toLock.Lock()
	defer toLock.Unlock()

	// delete current ownership
	transfer(previousTxId, transferTxId, transferBlockNumber, currentOwner, nil, balance)
}

// Transfer - transfer ownership
func Transfer(previousTxId merkle.Digest, transferTxId merkle.Digest, transferBlockNumber uint64, currentOwner *account.Account, newOwner *account.Account) {

	// ensure single threaded
	toLock.Lock()
	defer toLock.Unlock()

	transfer(previousTxId, transferTxId, transferBlockNumber, currentOwner, newOwner, 0)
}

// need to hold the lock before calling this
func transfer(previousTxId merkle.Digest, transferTxId merkle.Digest, transferBlockNumber uint64,
	currentOwner *account.Account, newOwner *account.Account, quantity uint64) {

	// get count for current owner record
	dKey := append(currentOwner.Bytes(), previousTxId[:]...)
	dCount := storage.Pool.OwnerTxIndex.Get(dKey)
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

		logger.Panic("ownership.Transfer: OwnerTxIndex database corrupt")
	}

	// delete the current owners records
	ownerData, err := GetOwnerData(previousTxId)
	if nil != err {
		logger.Criticalf("ownership.Transfer: invalid owner data for tx id: %s  error: %s", previousTxId, err)
		logger.Panic("ownership.Transfer: Ownership database corrupt")
	}

	oKey := append(currentOwner.Bytes(), dCount...)
	storage.Pool.OwnerList.Delete(oKey)
	storage.Pool.OwnerTxIndex.Delete(dKey)

	// and the old owner data
	storage.Pool.OwnerData.Delete(previousTxId[:])

	// if no new owner only above delete was needed
	if nil == newOwner && 0 == quantity {
		return
	}

	switch ownerData := ownerData.(type) {

	case *AssetOwnerData:

		// create a share - only from an asset
		if 0 != quantity {

			// convert initial quantity to 8 byte big endian
			quantityBytes := make([]byte, 8)
			binary.BigEndian.PutUint64(quantityBytes, quantity)

			// the ID of the share is the issue id of the bitmark
			shareId := ownerData.issueTxId

			// the total quantity of this type of share
			shareData := append(quantityBytes, transferTxId[:]...)
			storage.Pool.Shares.Put(shareId[:], shareData)

			// initially total quantity goes to the creator
			fKey := append(currentOwner.Bytes(), shareId[:]...)
			storage.Pool.ShareQuantity.Put(fKey, quantityBytes)

			// convert to share and update
			newOwnerData := ShareOwnerData{
				transferBlockNumber: transferBlockNumber,
				issueTxId:           ownerData.issueTxId,
				issueBlockNumber:    ownerData.issueBlockNumber,
				assetId:             ownerData.assetId,
			}
			create(transferTxId, newOwnerData, currentOwner)
			return
		}

		// otherwise create new ownership record
		ownerData.transferBlockNumber = transferBlockNumber
		create(transferTxId, ownerData, newOwner)

	case *BlockOwnerData:
		// create a share - only from an asset
		if 0 != quantity {

			// panic if not an asset (this should have been checked earlier)
			logger.Criticalf("ownership.Transfer: ownerData for key: %x is not an asset", oKey)
			logger.Panic("ownership.Transfer: Ownership database corrupt")
		}

		// otherwise create new ownership record
		ownerData.transferBlockNumber = transferBlockNumber
		create(transferTxId, ownerData, newOwner)

	case *ShareOwnerData:

		// create a share - only from an asset
		if 0 != quantity {

			// panic if not an asset (this should have been checked earlier)
			logger.Criticalf("ownership.Transfer: ownerData for key: %x is not an asset", oKey)
			logger.Panic("ownership.Transfer: Ownership database corrupt")
		}

		// Note: only called on delete (block/store.go prevents share back to asset)

		// convert to transfer and update
		newOwnerData := AssetOwnerData{
			transferBlockNumber: transferBlockNumber,
			issueTxId:           ownerData.issueTxId,
			issueBlockNumber:    ownerData.issueBlockNumber,
			assetId:             ownerData.assetId,
		}
		create(transferTxId, newOwnerData, currentOwner)

	default:
		// panic if not an asset (this should have been checked earlier)
		logger.Criticalf("ownership.Transfer: unhandled owner data type: %+v", ownerData)
		logger.Panic("ownership.Transfer: missing owner data handler")
	}
}

// internal creation routine, must be called with lock held
// adds items to owner's list of properties and stores the owner data
func create(txId merkle.Digest, ownerData OwnerData, owner *account.Account) {

	// increment the count for owner
	nKey := owner.Bytes()
	count := storage.Pool.OwnerNextCount.Get(nKey)
	if nil == count {
		count = []byte{0, 0, 0, 0, 0, 0, 0, 0}
	} else if uint64ByteSize != len(count) {
		logger.Panic("OwnerNextCount database corrupt")
	}
	newCount := make([]byte, uint64ByteSize)
	binary.BigEndian.PutUint64(newCount, binary.BigEndian.Uint64(count)+1)
	storage.Pool.OwnerNextCount.Put(nKey, newCount)

	// write to the owner list
	oKey := append(owner.Bytes(), count...)
	storage.Pool.OwnerList.Put(oKey, txId[:])

	// write new index record
	dKey := append(owner.Bytes(), txId[:]...)
	storage.Pool.OwnerTxIndex.Put(dKey, count)

	// save owner data record
	storage.Pool.OwnerData.Put(txId[:], ownerData.Pack())
}

// CreateAsset - create the ownership record for an asset
func CreateAsset(issueTxId merkle.Digest, issueBlockNumber uint64, assetId transactionrecord.AssetIdentifier, newOwner *account.Account) {
	// ensure single threaded
	toLock.Lock()
	defer toLock.Unlock()

	newData := &AssetOwnerData{
		transferBlockNumber: issueBlockNumber,
		issueTxId:           issueTxId,
		issueBlockNumber:    issueBlockNumber,
		assetId:             assetId,
	}

	// store to database
	create(issueTxId, newData, newOwner)
}

// CreateBlock - create the ownership record for a block
func CreateBlock(issueTxId merkle.Digest, blockNumber uint64, newOwner *account.Account) {
	// ensure single threaded
	toLock.Lock()
	defer toLock.Unlock()

	newData := &BlockOwnerData{
		transferBlockNumber: blockNumber,
		issueTxId:           issueTxId,
		issueBlockNumber:    blockNumber,
	}

	// store to database
	create(issueTxId, newData, newOwner)
}

// OwnerOf - find the owner of a specific transaction
func OwnerOf(txId merkle.Digest) (uint64, *account.Account) {

	blockNumber, packed := storage.Pool.Transactions.GetNB(txId[:])
	if nil == packed {
		return 0, nil
	}

	transaction, _, err := transactionrecord.Packed(packed).Unpack(mode.IsTesting())
	logger.PanicIfError("ownership.OwnerOf", err)

	switch tx := transaction.(type) {
	case *transactionrecord.BitmarkIssue:
		return blockNumber, tx.Owner

	case *transactionrecord.BitmarkTransferUnratified:
		return blockNumber, tx.Owner

	case *transactionrecord.BitmarkTransferCountersigned:
		return blockNumber, tx.Owner

	case *transactionrecord.BlockFoundation:
		return blockNumber, tx.Owner

	case *transactionrecord.BlockOwnerTransfer:
		return blockNumber, tx.Owner

	default:
		logger.Panicf("block.OwnerOf: incorrect transaction: %v", transaction)
		return 0, nil
	}
}

// CurrentlyOwns - check owner currently owns this transaction id
func CurrentlyOwns(owner *account.Account, txId merkle.Digest) bool {
	dKey := append(owner.Bytes(), txId[:]...)
	return storage.Pool.OwnerTxIndex.Has(dKey)
}
