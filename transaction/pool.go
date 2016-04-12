// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package transaction

import (
	"bytes"
	"encoding/binary"
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/counter"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/pool"
	"github.com/bitmark-inc/logger"
	"sync"
	"time"
)

// transaction pool protected variables
var transactionPool struct {
	sync.RWMutex // to allow locking

	// set once during initilise - all routines must error if this is false
	initialised bool

	// channel for logging
	log *logger.L

	// data storage pools
	dataPool  *pool.Pool // raw transaction data
	statePool *pool.Pool // state: pending, verified or confirmed

	// state index pools
	pendingPool  *pool.Pool // index of pending payment
	verifiedPool *pool.Pool // index of payment verified so tx can be mined

	// global counts
	pendingCounter  counter.Counter
	verifiedCounter counter.Counter

	// index of assets
	assetPool *pool.Pool // all assets

	// owner index pools
	ownerCountPool  *pool.Pool // couters for assigning indexes
	ownershipPool   *pool.Pool // ownership data
	ownerDigestPool *pool.Pool // back link for deletions after transfer

	// counter for record index
	// used as index for the pending/verified pools
	indexCounter IndexCursor
}

// initialise the transaction data pool
func Initialise() {
	transactionPool.Lock()
	defer transactionPool.Unlock()
	// no need to start if already started
	if transactionPool.initialised {
		return
	}

	transactionPool.log = logger.New("transaction")
	transactionPool.log.Info("starting…")

	transactionPool.indexCounter = 0

	transactionPool.dataPool = pool.New(pool.TransactionData)
	transactionPool.statePool = pool.New(pool.TransactionState)

	transactionPool.pendingPool = pool.New(pool.PendingIndex)
	transactionPool.verifiedPool = pool.New(pool.VerifiedIndex)

	transactionPool.pendingCounter = 0
	transactionPool.verifiedCounter = 0

	transactionPool.assetPool = pool.New(pool.AssetData)

	transactionPool.ownerCountPool = pool.New(pool.OwnerCount)
	transactionPool.ownershipPool = pool.New(pool.Ownership)
	transactionPool.ownerDigestPool = pool.New(pool.OwnerDigest)

	// check if transactions are missing state
	dataCursor := transactionPool.dataPool.NewFetchCursor()
	abort := false
	for {
		// read blocks of records
		transactions, err := dataCursor.Fetch(100)
		fault.PanicIfError("transaction.Initialise: dataPool fetch", err)

		// if no more records exit loop
		if 0 == len(transactions) {
			break
		}

		for _, t := range transactions {
			txId := t.Key
			transactionPool.log.Tracef("check state for: %x", txId)

			if !transactionPool.statePool.Has(txId) {
				transactionPool.log.Criticalf("Initialise: missing state for TxId: %x", txId)
				abort = true
			}
			// rebuild asset index
			switch Packed(t.Value).Type() {
			case AssetDataTag:
				asset, err := Packed(t.Value).Unpack()
				fault.PanicIfError("transaction.Initialise: unpack asset failed", err)
				assetIndex := asset.(*AssetData).AssetIndex()
				transactionPool.assetPool.Add(assetIndex.Bytes(), txId)
			default:
			}
		}
	}
	if abort {
		fault.Panic("transaction.Initialise: missing transaction state")
	}

	// make sure mined status is correct
	lastBlock := block.Number()
	stateBuffer := make([]byte, 9)
	stateBuffer[0] = byte(ConfirmedTransaction)

	abort = false
	for n := uint64(2); n < lastBlock; n += 1 {
		transactionPool.log.Debugf("set confirmed from block: %d", n)
		packed := block.Get(n)
		if nil == packed {
			fault.Panicf("transaction.Initialise: missing block: %d", n)
		}
		var blk block.Block
		err := packed.Unpack(&blk)
		fault.PanicIfError("transaction.Initialise: block recovery failed, block unpack", err)

		difficulty.Current.SetBits(blk.Header.Bits.Bits())

		// rewrite as confirmed
		for _, txId := range blk.TxIds {
			indexBuffer := Link(txId).Bytes()
			if transactionPool.dataPool.Has(indexBuffer) {
				transactionPool.statePool.Add(indexBuffer, stateBuffer)
			} else {
				transactionPool.log.Criticalf("Initialise: missing tx: %#v", Link(txId))
				abort = true
			}
		}
	}
	if abort {
		fault.Panic("transaction.Initialise: missing transactions")
	}

	// check asset index
	assetCursor := transactionPool.assetPool.NewFetchCursor()
	for {
		// read blocks of records
		assets, err := assetCursor.Fetch(100)
		fault.PanicIfError("transaction.Initialise: assetPool fetch", err)

		// if no more records exit loop
		if 0 == len(assets) {
			break
		}

		for _, e := range assets {
			if !transactionPool.dataPool.Has(e.Value) {
				var txId Link
				err := LinkFromBytes(&txId, e.Value)
				fault.PanicIfError("transaction.Initialise: failed to convert link", err)
				var assetIndex AssetIndex
				err = AssetIndexFromBytes(&assetIndex, e.Key)
				fault.PanicIfError("transaction.Initialise: failed to convert asset index", err)

				transactionPool.log.Warnf("Initialise: delete index: %#v  with missing tx: %#v", assetIndex, txId)
				transactionPool.assetPool.Remove(e.Key)

				abort = true
			}
		}
	}
	if abort {
		fault.Panic("transaction.Initialise: missing asset transactions: e.g. an expiry failed to delete asset index record")
	}

	// rebuild indexes
	// from pool/names.go
	//   S<tx-digest> - state: byte[expired(E), pending(P), verified(V), confirmed(C)] ++ int64[the U/V table count value]
	//   U<count>     - transaction-digest ++ int64[timestamp] (pending unverified transactions waiting for payment)
	stateCursor := transactionPool.statePool.NewFetchCursor()
	for {
		// read blocks of records
		state, err := stateCursor.Fetch(100)
		fault.PanicIfError("transaction.Initialise: statePool fetch", err)

		// if no more records exit loop
		if 0 == len(state) {
			break
		}

		// uint64 timestamp
		timestamp := uint64(time.Now().UTC().Unix())

		for _, e := range state {
			theState := State(e.Value[0])

			txId := e.Key
			indexBuffer := e.Value[1:]

			if !transactionPool.dataPool.Has(txId) {
				var txId Link
				err := LinkFromBytes(&txId, e.Value)
				fault.PanicIfError("transaction.Initialise: failed to convert link", err)

				transactionPool.log.Criticalf("Initialise: missing data for txid: %#v  with state: %#v", txId, theState)
				abort = true
			}

			transactionPool.log.Debugf("rebuild: %q %x", theState, txId)

			switch theState {

			case PendingTransaction:
				transactionPool.pendingCounter.Increment()
				// ensure an old timestamp is not updated
				if !transactionPool.pendingPool.Has(indexBuffer) {
					// Link ++ int64[timestamp]
					pendingData := make([]byte, LinkSize+8)
					copy(pendingData, txId)
					binary.BigEndian.PutUint64(pendingData[LinkSize:], timestamp)
					transactionPool.pendingPool.Add(indexBuffer, pendingData)
				}
				transactionPool.verifiedPool.Remove(indexBuffer)

			case VerifiedTransaction:
				transactionPool.pendingPool.Remove(indexBuffer)
				transactionPool.verifiedPool.Add(indexBuffer, txId)
				transactionPool.verifiedCounter.Increment()

			case ConfirmedTransaction:
				transactionPool.pendingPool.Remove(indexBuffer)
				transactionPool.verifiedPool.Remove(indexBuffer)

			default:
				// error represents an unexpected state of a transaction
				fault.Panicf("transaction.Initialise: invalid state: %s  for: %v", theState, txId)

			}
		}
	}
	if abort {
		fault.Panic("transaction.Initialise: missing data for transactions: e.g. an expiry failed to delete data record")
	}

	transactionPool.log.Debugf("pending count: %d", transactionPool.pendingCounter.Uint64())
	transactionPool.log.Debugf("verified count: %d", transactionPool.verifiedCounter.Uint64())

	// drop expired/non-existant pending
	pendingCursor := transactionPool.pendingPool.NewFetchCursor()
	for {
		records, err := pendingCursor.Fetch(100)
		fault.PanicIfError("transaction.Initialise: pendingPool fetch", err)

		// if no more records exit loop
		if 0 == len(records) {
			break
		}

		for _, record := range records {
			state := transactionPool.statePool.Get(record.Value[:LinkSize])
			if nil == state || PendingTransaction != State(state[0]) {
				transactionPool.pendingPool.Remove(record.Key)
				transactionPool.pendingCounter.Decrement()
			}
		}
	}

	transactionPool.log.Debugf("pending count (after drop): %d", transactionPool.pendingCounter.Uint64())

	// setup the cursor value to highest key value from either index
	if e, found := transactionPool.pendingPool.LastElement(); found {
		n := binary.BigEndian.Uint64(e.Key)
		if n > uint64(transactionPool.indexCounter) {
			transactionPool.indexCounter = IndexCursor(n)
		}
	}
	if e, found := transactionPool.verifiedPool.LastElement(); found {
		n := binary.BigEndian.Uint64(e.Key)
		if n > uint64(transactionPool.indexCounter) {
			transactionPool.indexCounter = IndexCursor(n)
		}
	}

	// initialisation is complete
	transactionPool.initialised = true
}

// finalise - flush unsaved data
func Finalise() {
	transactionPool.dataPool.Flush()
	transactionPool.statePool.Flush()
	transactionPool.pendingPool.Flush()
	transactionPool.verifiedPool.Flush()
	transactionPool.assetPool.Flush()
	transactionPool.log.Info("shutting down…")
	transactionPool.log.Flush()
}

// sanpshot of counts
func ReadCounters(pending *uint64, verified *uint64) {
	transactionPool.RLock()
	defer transactionPool.RUnlock()
	*pending = transactionPool.pendingCounter.Uint64()
	*verified = transactionPool.verifiedCounter.Uint64()
}

// write a transaction
//
// returns:
//   whether the values was added (false => already exists)
//   the ID of the transaction
//
// this enters the transaction as an pending new transaction
func (data Packed) Write(link *Link, overwriteAssetIndex bool) error {

	*link = data.MakeLink()
	txId := link.Bytes()

	transactionPool.Lock()
	defer transactionPool.Unlock()

	// if in overwrite mode, skip the exists check
	if !overwriteAssetIndex {
		if transactionPool.statePool.Has(txId) {
			return fault.ErrTransactionAlreadyExists
		}
	}

	// make a timestamp
	timestamp := uint64(time.Now().UTC().Unix()) // int64 timestamp

	// ensure transaction data is valid
	tx, err := data.Unpack()
	if nil != err {
		fault.PanicIfError("transaction.write unpack", err)

		return err // not reached
	}

	// batch updates
	batch := pool.NewBatch()
	defer batch.Commit()

	deletableTxId := []byte(nil)

	switch tx.(type) {
	case *AssetData:
		asset := tx.(*AssetData)
		assetIndex := asset.AssetIndex().Bytes()
		altTxId := transactionPool.assetPool.Get(assetIndex)
		transactionPool.log.Debugf("txid: %x  new transaction id: %x", txId, altTxId)
		if nil != altTxId {
			// determine link for pre-existing version of the same asset
			err := LinkFromBytes(link, altTxId)
			fault.PanicIfError("transaction.write asset", err)
			if !overwriteAssetIndex || bytes.Equal(txId, altTxId) {
				return fault.ErrTransactionAlreadyExists
			}
			deletableTxId = altTxId
			transactionPool.log.Debugf("deletable transaction id: %x", deletableTxId)
		}

	case *BitmarkIssue:
		if overwriteAssetIndex {
			transactionPool.log.Critical("cannot overwrite a non-asset")
		}

		transfer := tx.(*BitmarkIssue)

		// previous record
		assetIndex := transfer.AssetIndex.Bytes()

		// must link to an Asset
		assetLink := transactionPool.assetPool.Get(assetIndex)
		if nil == assetLink {
			transactionPool.log.Warnf("write tx, issue asset: %x", assetIndex)
			return fault.ErrAssetNotFound
		}

		// check asset
		assetState := transactionPool.statePool.Get(assetLink)
		if nil == assetState {
			dataFound := transactionPool.dataPool.Has(assetLink)
			transactionPool.log.Criticalf("write tx, asset Index: %x", assetIndex)
			transactionPool.log.Criticalf("write tx, asset tx id: %x", assetLink)
			transactionPool.log.Criticalf("write tx, data found:  %v", dataFound)
			transactionPool.log.Criticalf("write tx, state found: %v", assetState)
			fault.Panicf("write tx, no asset state for id: %x", assetLink)
			return fault.ErrAssetNotFound // not reached
		}

		// if asset is pending update timestamp and write back to give a longer expiry
		if PendingTransaction == State(assetState[0]) {
			data := transactionPool.pendingPool.Get(assetState[1:])
			if nil == data {
				fault.Panicf("write tx, no asset pending state for assetIndex: %x", assetIndex)
				return fault.ErrAssetNotFound // not reached
			}

			binary.BigEndian.PutUint64(data[LinkSize:], timestamp)

			batch.Add(transactionPool.pendingPool, assetState[1:], data)
		}

	default:
		if overwriteAssetIndex {
			transactionPool.log.Critical("cannot overwrite a non-asset")
		}
	}

	// create the index count in big endian order so
	// iterator on the index will return items in the
	// order they were entered
	indexBuffer := transactionPool.indexCounter.NextBytes()

	// first byte is state, next 8 bytes are big endian pending index
	stateBuffer := make([]byte, 9)
	stateBuffer[0] = byte(PendingTransaction)
	copy(stateBuffer[1:], indexBuffer)

	// update counter
	transactionPool.pendingCounter.Increment()

	// Link ++ int64[timestamp]
	pendingData := make([]byte, LinkSize+8)
	copy(pendingData, txId)
	binary.BigEndian.PutUint64(pendingData[LinkSize:], timestamp)

	// store in database
	batch.Add(transactionPool.dataPool, txId, data)
	batch.Add(transactionPool.statePool, txId, stateBuffer)
	batch.Add(transactionPool.pendingPool, indexBuffer, pendingData)
	switch tx.(type) {
	case *AssetData:
		asset := tx.(*AssetData)
		assetIndex := asset.AssetIndex().Bytes()
		batch.Add(transactionPool.assetPool, assetIndex, txId)
		// clean up unused asset tx/state/indexes now the correct one has be stored
		if nil != deletableTxId {
			transactionPool.log.Debugf("replacement  txid: %x", txId)
			transactionPool.log.Debugf("delete asset txid: %x", deletableTxId)
			batch.Remove(transactionPool.dataPool, deletableTxId)
			batch.Remove(transactionPool.statePool, deletableTxId)

			// need to remove from pending and/or verified if they exist and match
			if oldAssetState := transactionPool.statePool.Get(deletableTxId); nil != oldAssetState {
				indexBuffer := oldAssetState[1:] // get counter value
				if pending := transactionPool.pendingPool.Get(indexBuffer); nil != pending {
					if bytes.Equal(pending[:LinkSize], deletableTxId) {
						transactionPool.log.Debugf("delete pending: %x", indexBuffer)
						batch.Remove(transactionPool.pendingPool, indexBuffer)
						transactionPool.pendingCounter.Decrement()
					}
				}
				if verified := transactionPool.verifiedPool.Get(indexBuffer); nil != verified {
					if bytes.Equal(verified[:LinkSize], deletableTxId) {
						transactionPool.log.Debugf("delete verified: %x", indexBuffer)
						batch.Remove(transactionPool.verifiedPool, indexBuffer)
						transactionPool.verifiedCounter.Decrement()
					}
				}
			}
		}

	default:
	}

	transactionPool.log.Debugf("new transaction id: %x  data: %x", txId, data)

	return nil
}

// read a transaction
//
// returns:
//   state of record - see the const ExpiredTransaction,... above
//   record data as Packed type - just a byte slice
//   true if data was found
func (link Link) Read() (State, Packed, bool) {
	id := link.Bytes()
	state := transactionPool.statePool.Get(id)
	if nil == state {
		return ExpiredTransaction, nil, false
	}

	result := transactionPool.dataPool.Get(id)
	if nil == result {
		return ExpiredTransaction, nil, false
	}
	return State(state[0]), result, true
}

// state of a transaction
//
// returns:
//   state of record - see the const ExpiredTransaction,... above
//   true if data was found
func (link Link) GetState() (State, bool) {
	id := link.Bytes()
	state := transactionPool.statePool.Get(id)
	if nil == state {
		return ExpiredTransaction, false
	}
	return State(state[0]), true
}

// read an Asset from its assetIndex
//
// returns:
//   state of record - see the const ExpiredTransaction,... above
//   transaction ID - can be used in link.READ
//   true if data was found
func (asset AssetIndex) Read() (State, Link, bool) {
	id := transactionPool.assetPool.Get(asset.Bytes())
	if nil == id {
		return ExpiredTransaction, Link{}, false
	}

	state := transactionPool.statePool.Get(id)
	if nil == state {
		return ExpiredTransaction, Link{}, false
	}

	var link Link
	err := LinkFromBytes(&link, id)
	if nil != err {
		fault.PanicWithError("asset.Read link conversion failed", err)
	}
	return State(state[0]), link, true
}

// see if allowed to transfer ownership
func (link Link) IsOwner(address *Address) bool {
	ownerDigestKey := append(address.PublicKeyBytes(), link.Bytes()...)
	return transactionPool.ownerDigestPool.Has(ownerDigestKey)
}

// must be called with locked mutex
func setAsset(batch *pool.Batch, assetNewState State, timestamp uint64, unpackedTransaction interface{}) {
	// if not a bitmark issue record the nothing to do
	issue, ok := unpackedTransaction.(*BitmarkIssue)
	if !ok {
		return
	}

	assetIndex := issue.AssetIndex.Bytes()

	// fetch the TxId corresponding to the asset
	assetTxId := transactionPool.assetPool.Get(assetIndex)
	if nil == assetTxId {
		fault.PanicWithError("transaction.SetState", fault.ErrLinkNotFound)
	}

	assetOldState, assetOldIndex := getStateIndex(assetTxId)
	if !assetOldState.CanChangeTo(assetNewState) {
		return
	}

	switch assetNewState {
	case ExpiredTransaction:
	case PendingTransaction:
	case VerifiedTransaction:
		setVerified(batch, assetOldState, assetOldIndex, assetTxId, timestamp)
	case ConfirmedTransaction:
		// fetch and decode the asset transaction
		rawTx := transactionPool.dataPool.Get(assetTxId)
		if nil == rawTx {
			fault.Panicf("transaction.setAsset: missing transaction for asset id: %x", assetTxId)
		}
		unpackedAssetTransaction, err := Packed(rawTx).Unpack()
		fault.PanicIfError("transaction.SetState: unpack", err)

		setConfirmed(batch, assetOldState, assetOldIndex, assetTxId, unpackedAssetTransaction)
	default:
	}

}

// must be called with locked mutex
func setVerified(batch *pool.Batch, oldState State, oldIndex []byte, txId []byte, timestamp uint64) bool {

	// create the index count in big endian order so
	// iterator on the index will return items in the
	// order they were entered
	indexBuffer := transactionPool.indexCounter.NextBytes()

	// first byte is state, next 8 bytes are big endian pending index
	stateBuffer := make([]byte, 9)
	stateBuffer[0] = byte(VerifiedTransaction)
	copy(stateBuffer[1:], indexBuffer)

	// rewrite as available
	batch.Add(transactionPool.statePool, txId, stateBuffer)

	// Link ++ int64[timestamp]
	verifiedData := make([]byte, LinkSize+8)
	copy(verifiedData, txId)
	binary.BigEndian.PutUint64(verifiedData[LinkSize:], timestamp)

	// create verified
	batch.Add(transactionPool.verifiedPool, indexBuffer, verifiedData)
	transactionPool.verifiedCounter.Increment()

	// remove previous state
	switch oldState {
	case PendingTransaction:
		batch.Remove(transactionPool.pendingPool, oldIndex)
		transactionPool.pendingCounter.Decrement()

	default: // should not happen
		fault.Panicf("transaction.setVerified: invalid old state: %s", oldState)
	}

	return true
}

// must be called with locked mutex
func setConfirmed(batch *pool.Batch, oldState State, oldIndex []byte, txId []byte, unpackedTransaction interface{}) {

	// first byte is state, next 8 bytes are big endian zero (for compatibility of other states)
	stateBuffer := make([]byte, 9)
	stateBuffer[0] = byte(ConfirmedTransaction)

	// rewrite as confirmed
	batch.Add(transactionPool.statePool, txId, stateBuffer)

	// decode the transaction
	switch unpackedTransaction.(type) {
	case *AssetData:
		asset := unpackedTransaction.(*AssetData)
		assetIndex := NewAssetIndex([]byte(asset.Fingerprint)).Bytes()
		batch.Add(transactionPool.assetPool, assetIndex, txId)

	case *BitmarkIssue:
		transfer := unpackedTransaction.(*BitmarkIssue)

		// previous record
		assetIndex := transfer.AssetIndex.Bytes()

		// must link to an Asset
		assetDataLink := transactionPool.assetPool.Get(assetIndex)
		if nil == assetDataLink {
			fault.PanicWithError("transaction.setConfirmed: asset look up", fault.ErrLinkNotFound)
		}

		count := uint64(0)
		n := transactionPool.ownerCountPool.Get(transfer.Owner.PublicKeyBytes())
		if nil == n {
			n = make([]byte, 8)
		} else if len(n) == 8 {
			count = binary.BigEndian.Uint64(n)
		} else {
			fault.Panicf("transaction.setConfirmed: invalid n : %x", n)
		}
		count += 1
		binary.BigEndian.PutUint64(n, count)
		batch.Add(transactionPool.ownerCountPool, transfer.Owner.PublicKeyBytes(), n)

		ownershipKey := append(transfer.Owner.PublicKeyBytes(), n...)
		ownershipData := append([]byte{}, txId...)     // also the issue digest
		ownershipData = append(ownershipData, txId...) // the issue digest
		ownershipData = append(ownershipData, assetDataLink...)
		batch.Add(transactionPool.ownershipPool, ownershipKey, ownershipData)

		ownerDigestKey := append(transfer.Owner.PublicKeyBytes(), txId...)
		batch.Add(transactionPool.ownerDigestPool, ownerDigestKey, n)

	case *BitmarkTransfer:
		transfer := unpackedTransaction.(*BitmarkTransfer)

		// previous record
		previousLink := transfer.Link.Bytes()
		previous := transactionPool.dataPool.Get(previousLink)
		if nil == previous {
			fault.Criticalf("transaction.setConfirmed: no previous transaction for: %v", previousLink)
			fault.PanicWithError("transaction.setConfirmed", fault.ErrLinkNotFound)
		}

		transaction, err := Packed(previous).Unpack()
		fault.PanicIfError("transaction.setConfirmed: unpack previous:", err)

		var previousOwner *Address
		switch transaction.(type) {
		case *BitmarkIssue:
			previousIssue := transaction.(*BitmarkIssue)
			previousOwner = previousIssue.Owner
		case *BitmarkTransfer:
			previousTransfer := transaction.(*BitmarkTransfer)
			previousOwner = previousTransfer.Owner
		default:
			fault.Panic("transaction.setConfirmed: no previous transaction is invalid")
		}

		// fetch previous owner data

		previousOwnerDigestKey := append(previousOwner.PublicKeyBytes(), previousLink...)
		previousN := transactionPool.ownerDigestPool.Get(previousOwnerDigestKey)
		if nil == previousN {
			fault.Panicf("Database corrupted: not found: previousOwnerDigestKey: %x", previousOwnerDigestKey)
		} else if len(previousN) != 8 {
			fault.Criticalf("Database corrupted: previousOwnerDigestKey: %x", previousOwnerDigestKey)
			fault.Panicf("Database corrupted: invalid previousN: %x", previousN)
		}

		previousOwnershipKey := append(previousOwner.PublicKeyBytes(), previousN...)
		previousOwnership := transactionPool.ownershipPool.Get(previousOwnershipKey)
		if nil == previousOwnership {
			fault.Panicf("Database corrupted: not found: previousOwnershipKey: %x", previousOwnershipKey)
		}

		//previousTxId := previousOwnership[:LinkSize]  // Verify this? should == previousLink
		issueTxId := previousOwnership[LinkSize : 2*LinkSize]
		assetTxId := previousOwnership[2*LinkSize:]

		// set new owner index number

		count := uint64(0)
		n := transactionPool.ownerCountPool.Get(transfer.Owner.PublicKeyBytes())
		if nil == n {
			n = make([]byte, 8)
		} else if len(n) == 8 {
			count = binary.BigEndian.Uint64(n)
		} else {
			fault.Panicf("transaction.setConfirmed: invalid n : %x", n)
		}
		count += 1
		binary.BigEndian.PutUint64(n, count)
		batch.Add(transactionPool.ownerCountPool, transfer.Owner.PublicKeyBytes(), n)

		// remove previous owner
		batch.Remove(transactionPool.ownershipPool, previousOwnershipKey)
		batch.Remove(transactionPool.ownerDigestPool, previousOwnerDigestKey)

		// save new owner
		ownershipKey := append(transfer.Owner.PublicKeyBytes(), n...)
		ownershipData := append([]byte{}, txId...)
		ownershipData = append(ownershipData, issueTxId...)
		ownershipData = append(ownershipData, assetTxId...)
		batch.Add(transactionPool.ownershipPool, ownershipKey, ownershipData)

		ownerDigestKey := append(transfer.Owner.PublicKeyBytes(), txId...)
		batch.Add(transactionPool.ownerDigestPool, ownerDigestKey, n)

	default:
		fault.Panicf("transaction.setConfirmed: unknown transaction type: %v", unpackedTransaction)
	}

	// decrement apropriate counter
	switch oldState {

	case PendingTransaction:
		batch.Remove(transactionPool.pendingPool, oldIndex)
		transactionPool.pendingCounter.Decrement()

	case VerifiedTransaction:
		batch.Remove(transactionPool.verifiedPool, oldIndex)
		transactionPool.verifiedCounter.Decrement()

	default:
		fault.Panicf("transaction.setConfirmed: invalid old state: %s", oldState)
	}
}

func getStateIndex(txId []byte) (oldState State, oldIndex []byte) {
	tempStateData := transactionPool.statePool.Get(txId)
	if nil == tempStateData {
		fault.Criticalf("transaction.getTx: cannot find txid: %x", txId)
		fault.Panic("transaction.getTx: missing transaction state")
	}

	// save state fields before the temp disappears
	oldState = State(tempStateData[0])
	oldIndex = make([]byte, 8)
	copy(oldIndex, tempStateData[1:])
	return
}

// set the state of a transaction
func (link Link) SetState(newState State) {
	transactionPool.Lock()
	defer transactionPool.Unlock()

	txId := link.Bytes()

	oldState, oldIndex := getStateIndex(txId)

	// if invalid change then ignore
	if !oldState.CanChangeTo(newState) {
		return
	}

	// batch the adds and removes
	batch := pool.NewBatch()
	defer batch.Commit()

	// fetch and decode the transaction
	rawTx := transactionPool.dataPool.Get(txId)
	if nil == rawTx {
		fault.Panicf("transaction.SetState: missing transaction for id: %#v", link)
	}
	unpackedTransaction, err := Packed(rawTx).Unpack()
	fault.PanicIfError("transaction.SetState: unpack", err)

	// make a timestamp
	timestamp := uint64(time.Now().UTC().Unix()) // uint64 timestamp

	// check allowable transitions
	// Asset:
	//   P   → E (after timeout. E is not saved, records are just removed)
	//   E   → P (when issue → P)
	//   E,P → V (when issue → V)
	//   *   → C (when issue → C)
	// Issue, Transfer:
	//   P   → E (after timeout. E is not saved, records are just removed)
	//   P   → V (when payment is in currency block with 'N' confirmations)
	//   V   → C (when miner has found block)

	// flag to indicate if transition was correct
	ok := false

	// transition from old state to new state
	switch oldState {

	case ExpiredTransaction:
		// should not happen
		fault.Panicf("transaction.SetState - expired tx id: %#v", txId)

	case PendingTransaction:
		// allowed transitions: expired, pending, verified
		switch newState {

		case ExpiredTransaction:
			switch unpackedTransaction.(type) {
			case *AssetData:
				// if tx is asset remove the asset index record
				asset := unpackedTransaction.(*AssetData)
				assetIndex := NewAssetIndex([]byte(asset.Fingerprint)).Bytes()
				batch.Remove(transactionPool.assetPool, assetIndex)
			default:
			}
			// delete all associated records
			batch.Remove(transactionPool.pendingPool, oldIndex)
			batch.Remove(transactionPool.statePool, txId)
			batch.Remove(transactionPool.dataPool, txId)

			transactionPool.pendingCounter.Decrement()
			ok = true

		case VerifiedTransaction:
			setVerified(batch, oldState, oldIndex, txId, timestamp)
			setAsset(batch, VerifiedTransaction, timestamp, unpackedTransaction)
			ok = true

		case ConfirmedTransaction:
			setConfirmed(batch, oldState, oldIndex, txId, unpackedTransaction)
			setAsset(batch, ConfirmedTransaction, timestamp, unpackedTransaction)
			ok = true

		default:
		}

	case VerifiedTransaction:
		// allowed transitions: confirmed
		switch newState {
		case ConfirmedTransaction:
			setConfirmed(batch, oldState, oldIndex, txId, unpackedTransaction)
			setAsset(batch, ConfirmedTransaction, timestamp, unpackedTransaction)
			ok = true

		default:
		}

	case ConfirmedTransaction:

	default:
	}

	// should not happen, code is broken - so panic
	if !ok {
		fault.Criticalf("changing state on txid: %#v", link)
		fault.Panicf("from: '%c'(%d)  to: '%c'(%d)  is forbidden", oldState, oldState, newState, newState)
	}
}

// see if a transaction already exists and compute its ID
//
// note this will panic if database inconsistancy is detected
func (data Packed) Exists() (Link, bool) {

	// if an asset then need a different check
	switch data.Type() {
	case AssetDataTag:
		asset, err := data.Unpack()
		fault.PanicIfError("transaction.pool.Exists: unpack asset error: %v", err)
		idBytes := transactionPool.assetPool.Get(asset.(*AssetData).AssetIndex().Bytes())
		var id Link
		err = LinkFromBytes(&id, idBytes)
		if nil != idBytes && nil != err {
			fault.Panicf("transaction.pool.Exists: database corruption detected cannot convert asset link: %x  error: %v", idBytes, err)
		}
		return id, nil != idBytes
	default:
	}

	id := data.MakeLink()
	found := transactionPool.dataPool.Has(id.Bytes())
	return id, found
}
