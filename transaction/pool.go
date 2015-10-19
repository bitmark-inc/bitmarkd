// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package transaction

import (
	"bytes"
	"encoding/binary"
	"github.com/bitmark-inc/bitmarkd/block"
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
	verifiedPool *pool.Pool // index of payment veiified so tx can be mined

	// global counts
	pendingCounter  ItemCounter
	verifiedCounter ItemCounter

	// index of assets
	assetPool *pool.Pool // all assets

	// owner index pools
	ownerPool *pool.Pool // index of leaf bitmark transfers == current owner

	// counter for record index
	// used as index for the pending/verified pools
	indexCounter IndexCursor
}

// initialise the transaction data pool
func Initialise(cacheSize int) {
	transactionPool.Lock()
	defer transactionPool.Unlock()

	// no need to start if already started
	if transactionPool.initialised {
		return
	}

	transactionPool.log = logger.New("transaction")
	transactionPool.log.Info("starting…")

	transactionPool.indexCounter = 0

	transactionPool.dataPool = pool.New(pool.TransactionData, cacheSize)
	transactionPool.statePool = pool.New(pool.TransactionState, cacheSize)

	transactionPool.pendingPool = pool.New(pool.PendingIndex, cacheSize)
	transactionPool.verifiedPool = pool.New(pool.VerifiedIndex, cacheSize)

	transactionPool.pendingCounter = 0
	transactionPool.verifiedCounter = 0

	transactionPool.assetPool = pool.New(pool.AssetData, cacheSize)

	transactionPool.ownerPool = pool.New(pool.OwnerIndex, cacheSize)

	startIndex := []byte{}

	// make sure mined status is correct
	lastBlock := block.Number()
	stateBuffer := make([]byte, 9)
	stateBuffer[0] = byte(ConfirmedTransaction)

	abort := false
	for n := uint64(2); n < lastBlock; n += 1 {
		transactionPool.log.Debugf("set confirmed from block: %d", n)
		packed, found := block.Get(n)
		if !found {
			fault.Panicf("transaction.Initialise: missing block: %d", n)
		}
		var blk block.Block
		err := packed.Unpack(&blk)
		fault.PanicIfError("transaction.Initialise: block recovery failed, block unpack", err)

		difficulty.Current.SetBits(blk.Header.Bits.Bits())

		// rewrite as confirmed
		for _, txId := range blk.TxIds {
			indexBuffer := Link(txId).Bytes()
			if _, found := transactionPool.dataPool.Get(indexBuffer); found {
				transactionPool.statePool.Add(indexBuffer, stateBuffer)
			} else {
				transactionPool.log.Criticalf("transaction.Initialise: missing tx: %#v", Link(txId))
				abort = true
			}
		}
	}
	if abort {
		fault.Panic("transaction.Initialise: missing transactions")
	}

	// rebuild indexes
	// from pool/names.go
	//   S<tx-digest> - state: byte[expired(E), pending(P), verified(V), confirmed(C)] ++ int64[the U/V table count value]
	//   U<count>     - transaction-digest ++ int64[timestamp] (pending unverified transactions waiting for payment)
	for {
		// read blocks of records
		state, err := transactionPool.statePool.Fetch(startIndex, 100)
		fault.PanicIfError("transaction.Initialise: statePool fetch", err)

		// if no more records exit loop
		n := len(state)
		if n <= 1 {
			break
		}

		// uint64 timestamp
		timestamp := uint64(time.Now().UTC().Unix())

		for _, e := range state {
			theState := State(e.Value[0])

			txId := e.Key
			indexBuffer := e.Value[1:]

			transactionPool.log.Debugf("rebuild: %q %x", theState, txId)

			switch theState {

			case PendingTransaction:
				transactionPool.pendingCounter.Increment()
				// ensure an old timestamp is not updated
				if _, found := transactionPool.pendingPool.Get(indexBuffer); !found {
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
		startIndex = state[n-1].Key
	}

	// drop expired pending
	startIndex = []byte{}
	for {
		records, err := transactionPool.pendingPool.Fetch(startIndex, 100)
		fault.PanicIfError("transaction.Initialise: pendingPool fetch", err)

		// if no more records exit loop
		n := len(records)
		if n <= 1 {
			break
		}

		for _, record := range records {
			if state, found := transactionPool.statePool.Get(record.Value[:LinkSize]); found {
				if PendingTransaction != State(state[0]) {
					transactionPool.pendingPool.Remove(record.Key)
				}
			}
		}
		startIndex = records[n-1].Key
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
	transactionPool.ownerPool.Flush()
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
func (data Packed) Write(link *Link) error {

	*link = data.MakeLink()
	txId := link.Bytes()

	transactionPool.Lock()
	defer transactionPool.Unlock()

	if _, found := transactionPool.statePool.Get(txId); found {
		return fault.ErrTransactionAlreadyExists
	}

	// make a timestamp
	timestamp := uint64(time.Now().UTC().Unix()) // int64 timestamp

	// ensure transaction data is valid
	tx, err := data.Unpack()
	if nil != err {
		fault.PanicIfError("transaction.write unpack", err)

		return err // not reached
	}

	switch tx.(type) {
	case *AssetData:
		asset := tx.(*AssetData)
		assetIndex := asset.AssetIndex().Bytes()
		txId, found := transactionPool.assetPool.Get(assetIndex)
		if found {
			// determine link for pre-existing version of the same asset
			err := LinkFromBytes(link, txId)
			fault.PanicIfError("transaction.write asset", err)
			//return err // not reached
			return fault.ErrTransactionAlreadyExists
		}

	case *BitmarkIssue:
		transfer := tx.(*BitmarkIssue)

		// previous record
		assetIndex := transfer.AssetIndex.Bytes()

		// must link to an Asset
		previous, found := transactionPool.assetPool.Get(assetIndex)
		if !found {
			transactionPool.log.Warnf("write tx, issue asset: %x", assetIndex)
			return fault.ErrAssetNotFound
		}

		// split the record
		length := len(previous) - LinkSize
		//previousOwner := previous[:length]
		assetDataLink := previous[length:]

		// check asset
		assetState, found := transactionPool.statePool.Get(assetDataLink)
		if !found {
			fault.Panicf("write tx, no asset state for assetIndex: %x", assetIndex)
			return fault.ErrAssetNotFound // not reached
		}

		// if asset is pending update timestamp and write back to give a longer expiry
		if PendingTransaction == State(assetState[0]) {
			data, found := transactionPool.pendingPool.Get(assetState[1:])
			if !found {
				fault.Panicf("write tx, no asset pending state for assetIndex: %x", assetIndex)
				return fault.ErrAssetNotFound // not reached
			}

			binary.BigEndian.PutUint64(data[LinkSize:], timestamp)

			transactionPool.pendingPool.Add(assetState[1:], data)
		}

	default:
	}

	transactionPool.indexCounter += 1 // safe because mutex is locked
	// create the index count in big endian order so
	// iterator on the index will return items in the
	// order they were entered
	indexBuffer := transactionPool.indexCounter.Bytes()

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
	transactionPool.dataPool.Add(txId, data)
	transactionPool.statePool.Add(txId, stateBuffer)
	transactionPool.pendingPool.Add(indexBuffer, pendingData)
	switch tx.(type) {
	case *AssetData:
		asset := tx.(*AssetData)
		assetIndex := asset.AssetIndex().Bytes()
		transactionPool.assetPool.Add(assetIndex, txId)
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
	state, found := transactionPool.statePool.Get(id)
	if !found {
		return ExpiredTransaction, nil, false
	}

	result, found := transactionPool.dataPool.Get(id)
	if !found {
		return ExpiredTransaction, nil, false
	}
	return State(state[0]), result, true
}

// state of a transaction
//
// returns:
//   state of record - see the const ExpiredTransaction,... above
//   true if data was found
func (link Link) State() (State, bool) {
	id := link.Bytes()
	state, found := transactionPool.statePool.Get(id)
	if !found {
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
	id, found := transactionPool.assetPool.Get(asset.Bytes())
	if !found {
		return ExpiredTransaction, Link{}, false
	}

	state, found := transactionPool.statePool.Get(id)
	if !found {
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
	publicKeyAndAssetDataLink, found := transactionPool.ownerPool.Get(link.Bytes())
	if !found {
		return false
	}
	length := len(publicKeyAndAssetDataLink)
	publicKey := publicKeyAndAssetDataLink[:length-LinkSize]
	return bytes.Equal(publicKey, address.PublicKeyBytes())
}

// must be called with locked mutex
func setAsset(assetNewState State, timestamp uint64, unpackedTransaction interface{}) {
	// if not a bitmark issue record the nothing to do
	issue, ok := unpackedTransaction.(*BitmarkIssue)
	if !ok {
		return
	}

	assetIndex := issue.AssetIndex.Bytes()

	// fetch the TxId corresponding to the asset
	assetTxId, found := transactionPool.assetPool.Get(assetIndex)
	if !found {
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
		setVerified(assetOldState, assetOldIndex, assetTxId, timestamp)
	case ConfirmedTransaction:
		// fetch and decode the asset transaction
		rawTx, found := transactionPool.dataPool.Get(assetTxId)
		if !found {
			fault.Panicf("transaction.setAsset: missing transaction for asset id: %x", assetTxId)
		}
		unpackedAssetTransaction, err := Packed(rawTx).Unpack()
		fault.PanicIfError("transaction.SetState: unpack", err)

		setConfirmed(assetOldState, assetOldIndex, assetTxId, unpackedAssetTransaction)
	default:
	}

}

// must be called with locked mutex
func setVerified(oldState State, oldIndex []byte, txId []byte, timestamp uint64) bool {

	// create the index count in big endian order so
	// iterator on the index will return items in the
	// order they were entered
	indexBuffer := transactionPool.indexCounter.NextBytes()

	// first byte is state, next 8 bytes are big endian pending index
	stateBuffer := make([]byte, 9)
	stateBuffer[0] = byte(VerifiedTransaction)
	copy(stateBuffer[1:], indexBuffer)

	// rewrite as available
	transactionPool.statePool.Add(txId, stateBuffer)

	// Link ++ int64[timestamp]
	verifiedData := make([]byte, LinkSize+8)
	copy(verifiedData, txId)
	binary.BigEndian.PutUint64(verifiedData[LinkSize:], timestamp)

	// create verified
	transactionPool.verifiedPool.Add(indexBuffer, verifiedData)
	transactionPool.verifiedCounter.Increment()

	// remove previous state
	switch oldState {
	case PendingTransaction:
		transactionPool.pendingPool.Remove(oldIndex)
		transactionPool.pendingCounter.Decrement()

	default: // should not happen
		fault.Panicf("transaction.setVerified: invalid old state: %s", oldState)
	}

	return true
}

func setConfirmed(oldState State, oldIndex []byte, txId []byte, unpackedTransaction interface{}) {

	// first byte is state, next 8 bytes are big endian zero (for compatibility of other states)
	stateBuffer := make([]byte, 9)
	stateBuffer[0] = byte(ConfirmedTransaction)

	// rewrite as confirmed
	transactionPool.statePool.Add(txId, stateBuffer)

	// decode the transaction
	switch unpackedTransaction.(type) {
	case *AssetData:
		asset := unpackedTransaction.(*AssetData)
		assetIndex := NewAssetIndex([]byte(asset.Fingerprint)).Bytes()
		transactionPool.assetPool.Add(assetIndex, txId)

	case *BitmarkIssue:
		transfer := unpackedTransaction.(*BitmarkIssue)

		// previous record
		assetIndex := transfer.AssetIndex.Bytes()

		// must link to an Asset
		previous, found := transactionPool.assetPool.Get(assetIndex)
		if !found {
			fault.PanicWithError("transaction.setConfirmed", fault.ErrLinkNotFound)
		}

		// split the record
		length := len(previous) - LinkSize
		//previousOwner := previous[:length]
		assetDataLink := previous[length:]

		ownerData := append(transfer.Owner.PublicKeyBytes(), assetDataLink...)
		transactionPool.ownerPool.Add(txId, ownerData)

	case *BitmarkTransfer:
		transfer := unpackedTransaction.(*BitmarkTransfer)

		// previous record
		previousLink := transfer.Link.Bytes()
		previous, found := transactionPool.ownerPool.Get(previousLink)
		if !found {
			fault.PanicWithError("transaction.setConfirmed", fault.ErrLinkNotFound)
		}

		// split the record
		length := len(previous) - LinkSize
		previousOwner := previous[:length]
		assetDataLink := previous[length:]

		// avoid side effect modification of assetDataLink
		previousKey := make([]byte, 0, len(previousOwner)+LinkSize)
		previousKey = append(previousKey, previousOwner...)
		previousKey = append(previousKey, previousLink...)

		ownerData := append(transfer.Owner.PublicKeyBytes(), assetDataLink...)

		transactionPool.ownerPool.Remove(previousLink)
		transactionPool.ownerPool.Add(txId, ownerData)

	default:
		fault.Panicf("transaction.setConfirmed: unknown transaction type: %v", unpackedTransaction)
	}

	// decrement apropriate counter
	switch oldState {

	case PendingTransaction:
		transactionPool.pendingPool.Remove(oldIndex)
		transactionPool.pendingCounter.Decrement()

	case VerifiedTransaction:
		transactionPool.verifiedPool.Remove(oldIndex)
		transactionPool.verifiedCounter.Decrement()

	default:
		fault.Panicf("transaction.setConfirmed: invalid old state: %s", oldState)
	}
}

func getStateIndex(txId []byte) (oldState State, oldIndex []byte) {
	tempStateData, found := transactionPool.statePool.Get(txId)
	if !found {
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

	// if no change then ignore
	if oldState == newState {
		return
	}

	// fetch and decode the transaction
	rawTx, found := transactionPool.dataPool.Get(txId)
	if !found {
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

	// flag to indicate if transaition was correct
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
			// delete all associated records
			transactionPool.pendingPool.Remove(oldIndex)
			transactionPool.statePool.Remove(txId)
			transactionPool.dataPool.Remove(txId)

			transactionPool.pendingCounter.Decrement()
			ok = true

		case VerifiedTransaction:
			setVerified(oldState, oldIndex, txId, timestamp)
			setAsset(VerifiedTransaction, timestamp, unpackedTransaction)
			ok = true

		case ConfirmedTransaction:
			setConfirmed(oldState, oldIndex, txId, unpackedTransaction)
			setAsset(ConfirmedTransaction, timestamp, unpackedTransaction)
			ok = true

		default:
		}

	case VerifiedTransaction:
		// allowed transitions: confirmed
		switch newState {
		case ConfirmedTransaction:
			setConfirmed(oldState, oldIndex, txId, unpackedTransaction)
			setAsset(ConfirmedTransaction, timestamp, unpackedTransaction)
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

	id := data.MakeLink()
	result, found := transactionPool.dataPool.Get(id.Bytes())
	if !found {
		return id, false
	}
	if !bytes.Equal(data, result) {
		// hopefully this is never reached - if it does then log some data and panic
		fault.Panicf("transaction.pool.Exists: database corruption detected received tx: %x  local copy: %X", data, result)
	}

	// found the record
	return id, true
}
