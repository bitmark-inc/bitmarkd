// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package transaction

import (
	"bytes"
	"encoding/binary"
	"github.com/bitmark-inc/bitmarkd/block"
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
	statePool *pool.Pool // state: unpaid, available or mined

	// state index pools
	unpaidPool    *pool.Pool // index of unpaid
	availablePool *pool.Pool // index of available to be mined

	// store of assets
	assetPool *pool.Pool // all available assets

	// owner index pools
	ownerPool *pool.Pool // index of leaves bitmark transfer

	// counter for record index
	// used as index for the unpaidPool / availablePool
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

	transactionPool.unpaidPool = pool.New(pool.UnpaidIndex, cacheSize)
	transactionPool.availablePool = pool.New(pool.AvailableIndex, cacheSize)

	transactionPool.assetPool = pool.New(pool.AssetData, cacheSize)

	transactionPool.ownerPool = pool.New(pool.OwnerIndex, cacheSize)

	startIndex := []byte{}

	// make sure mined status is correct
	lastBlock := block.Number()
	stateBuffer := make([]byte, 9)
	stateBuffer[0] = byte(MinedTransaction)

	abort := false
	for n := uint64(2); n < lastBlock; n += 1 {
		transactionPool.log.Debugf("set mined from block: %d", n)
		packed, found := block.Get(n)
		if !found {
			fault.Criticalf("transaction block recovery failed, missing block: %d", n)
			fault.Panic("transaction block recovery failed")
		}
		var blk block.Block
		err := packed.Unpack(&blk)
		fault.PanicIfError("transaction block recovery failed, block unpack", err)

		// rewrite as mined
		for _, txId := range blk.TxIds {
			indexBuffer := Link(txId).Bytes()
			if _, found := transactionPool.dataPool.Get(indexBuffer); found {
				transactionPool.statePool.Add(indexBuffer, stateBuffer)
			} else {
				transactionPool.log.Criticalf("missing tx: %#v", Link(txId))
				fault.Criticalf("missing tx: %#v", Link(txId))
				abort = true
			}
		}
	}
	if abort {
		fault.Critical("Would panic in this case") // ***** REMOVE THIS *****
		//fault.Panic("missing transactions")
	}

	// rebuild indexes
loop:
	for {
		// read blocks of records
		state, err := transactionPool.statePool.Fetch(startIndex, 100)
		if nil != err {
			// error represents a database failure - panic
			fault.Criticalf("transaction.Initialise: statePool.Fetch failed, err = %v", err)
			fault.Panic("transaction.Initialise: failed")
		}

		// if no more records exit loop
		n := len(state)
		if n <= 1 {
			break loop
		}
		//   S<tx-digest>          - state: byte[expired(E), unpaid(U), available(A), mined(M)] ++ int64[the U/A table count value]

		// uint64 timestamp
		timestamp := uint64(time.Now().UTC().Unix())

		for _, e := range state {
			theState := State(e.Value[0])

			txId := e.Key
			indexBuffer := e.Value[1:]

			transactionPool.log.Debugf("rebuild: %q %x", theState, txId)

			switch theState {

			case UnpaidTransaction, WaitingIssueTransaction:
				// ensure an old timestamp is not updated
				if _, found := transactionPool.unpaidPool.Get(indexBuffer); !found {
					// Link ++ int64[timestamp]
					unpaidData := make([]byte, LinkSize+8)
					copy(unpaidData, txId)
					binary.BigEndian.PutUint64(unpaidData[LinkSize:], timestamp)
					transactionPool.unpaidPool.Add(indexBuffer, unpaidData)
				}
				transactionPool.availablePool.Remove(indexBuffer)

			case AvailableTransaction:
				transactionPool.availablePool.Add(indexBuffer, txId)
				transactionPool.unpaidPool.Remove(indexBuffer)

			default:
				transactionPool.unpaidPool.Remove(indexBuffer)
				transactionPool.availablePool.Remove(indexBuffer)
			}
		}
		startIndex = state[n-1].Key
	}

	transactionPool.initialised = true
}

// finalise - flush unsaved data
func Finalise() {
	transactionPool.dataPool.Flush()
	transactionPool.statePool.Flush()
	transactionPool.unpaidPool.Flush()
	transactionPool.availablePool.Flush()
	transactionPool.assetPool.Flush()
	transactionPool.ownerPool.Flush()
	transactionPool.log.Info("shutting down…")
	transactionPool.log.Flush()
}

// write a transaction
//
// returns:
//   whether the values was added (false => already exists)
//   the ID of the transaction
//
// this enters the transaction as an unpaid new transaction
func (data Packed) Write(link *Link) error {

	*link = data.MakeLink()
	txId := link.Bytes()

	transactionPool.Lock()
	defer transactionPool.Unlock()

	if _, found := transactionPool.statePool.Get(txId); !found {

		// initial state
		startingState := UnpaidTransaction

		// make a timestamp
		timestamp := uint64(time.Now().UTC().Unix()) // int64 timestamp

		// check for duplicate asset and return previous transaction id
		tx, err := data.Unpack()
		if nil != err {
			transactionPool.log.Criticalf("write tx, unpack error: %v", err)
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
				transactionPool.log.Criticalf("write tx, unpack error: %v", err)
				fault.PanicIfError("transaction.write link from bytes", err)
				return err // not reached
			}
			startingState = WaitingIssueTransaction

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

			// determine if asset is in waiting state
			assetState, found := transactionPool.statePool.Get(assetDataLink)
			if !found {
				transactionPool.log.Criticalf("write tx, no asset state for assetIndex: %x", assetIndex)
				fault.Panic("transaction.write (no asset state)")
				return fault.ErrAssetNotFound // not reached
			}

			// if waiting update timestamp and write back
			if WaitingIssueTransaction == State(assetState[0]) {
				data, found := transactionPool.unpaidPool.Get(assetState[1:])
				if !found {
					transactionPool.log.Criticalf("write tx, no asset unpaid state for assetIndex: %x", assetIndex)
					fault.Panic("transaction.write (no asset unpaid state)")
					return fault.ErrAssetNotFound // not reached
				}

				binary.BigEndian.PutUint64(data[LinkSize:], timestamp)

				transactionPool.unpaidPool.Add(assetState[1:], data)
			}

		default:
		}

		transactionPool.indexCounter += 1 // safe because mutex is locked
		// create the index count in big endian order so
		// iterator on the index will return items in the
		// order they were entered
		indexBuffer := transactionPool.indexCounter.Bytes()

		// first byte is state, next 8 bytes are big endian unpaid index
		stateBuffer := make([]byte, 9)
		stateBuffer[0] = byte(startingState)
		copy(stateBuffer[1:], indexBuffer)

		// Link ++ int64[timestamp]
		unpaidData := make([]byte, LinkSize+8)
		copy(unpaidData, txId)
		binary.BigEndian.PutUint64(unpaidData[LinkSize:], timestamp)

		// store in database
		transactionPool.statePool.Add(txId, stateBuffer)
		transactionPool.unpaidPool.Add(indexBuffer, unpaidData)
		transactionPool.dataPool.Add(txId, data)
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
	return fault.ErrTransactionAlreadyExists
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

// set the state of a transaction
func (link Link) SetState(newState State) {
	transactionPool.Lock()
	defer transactionPool.Unlock()

	txId := link.Bytes()
	tempStateData, found := transactionPool.statePool.Get(txId)
	if !found {
		fault.Criticalf("SetState: cannot find txid: %#v", link)
		fault.Panic("SetState counld not find transaction")
	}

	// save state fields before the temp disappears
	oldState := State(tempStateData[0])
	oldIndex := make([]byte, 8)
	copy(oldIndex, tempStateData[1:])

	// if no change then ignore
	if oldState == newState {
		return
	}

	// check allowable transitions
	ok := false
switchOldState:
	switch oldState {

	case ExpiredTransaction:
		// should not happen

	case UnpaidTransaction:
		switch newState {
		case AvailableTransaction:
			transactionPool.indexCounter += 1 // safe because mutex is locked
			// create the index count in big endian order so
			// iterator on the index will return items in the
			// order they were entered
			indexBuffer := transactionPool.indexCounter.Bytes()

			// first byte is state, next 8 bytes are big endian unpaid index
			stateBuffer := make([]byte, 9)
			stateBuffer[0] = byte(AvailableTransaction)
			copy(stateBuffer[1:], indexBuffer)

			// rewrite as available
			transactionPool.statePool.Add(txId, stateBuffer)

			// create available - remove unpaid
			transactionPool.availablePool.Add(indexBuffer, txId)
			transactionPool.unpaidPool.Remove(oldIndex)
			ok = true

		case ExpiredTransaction:
			// delete all associated records
			transactionPool.unpaidPool.Remove(oldIndex)
			transactionPool.statePool.Remove(txId)
			transactionPool.dataPool.Remove(txId)
			ok = true
		default:
		}

	case WaitingIssueTransaction, AvailableTransaction:
		switch newState {
		case ExpiredTransaction:
			if WaitingIssueTransaction != oldState {
				break switchOldState
			}
			// fetch and decode the transaction
			rawTx, found := transactionPool.dataPool.Get(txId)
			if !found {
				fault.Criticalf("transaction.SetState - missing transaction for id: %#v", link)
				fault.Panic("transaction.SetState - missing transaction")
			}
			record, err := Packed(rawTx).Unpack()
			fault.PanicIfError("transaction.SetState", err)

			switch record.(type) {
			case *AssetData:
				asset := record.(*AssetData)
				assetIndex := NewAssetIndex([]byte(asset.Fingerprint)).Bytes()
				transactionPool.assetPool.Remove(assetIndex)

				// delete all associated records
				transactionPool.unpaidPool.Remove(oldIndex)
				transactionPool.statePool.Remove(txId)
				transactionPool.dataPool.Remove(txId)
				ok = true
			}

		case MinedTransaction:

			// first byte is state, next 8 bytes are big endian zero (for compatibility of other states)
			stateBuffer := make([]byte, 9)
			stateBuffer[0] = byte(MinedTransaction)

			// rewrite as mined
			transactionPool.statePool.Add(txId, stateBuffer)

			// delete unpaid/available
			transactionPool.unpaidPool.Remove(oldIndex)
			transactionPool.availablePool.Remove(oldIndex)

			// fetch and decode the transaction
			rawTx, found := transactionPool.dataPool.Get(txId)
			if !found {
				fault.Criticalf("transaction.SetState - missing transaction for id: %#v", link)
				fault.Panic("transaction.SetState - missing transaction")
			}
			record, err := Packed(rawTx).Unpack()
			fault.PanicIfError("transaction.SetState", err)

			switch record.(type) {
			case *AssetData:
				asset := record.(*AssetData)
				assetIndex := NewAssetIndex([]byte(asset.Fingerprint)).Bytes()
				transactionPool.assetPool.Add(assetIndex, txId)

			case *BitmarkIssue:
				transfer := record.(*BitmarkIssue)

				// previous record
				assetIndex := transfer.AssetIndex.Bytes()

				// must link to an Asset
				previous, found := transactionPool.assetPool.Get(assetIndex)
				if !found {
					fault.PanicWithError("transaction.SetState", fault.ErrLinkNotFound)
				}

				// split the record
				length := len(previous) - LinkSize
				//previousOwner := previous[:length]
				assetDataLink := previous[length:]

				ownerData := append(transfer.Owner.PublicKeyBytes(), assetDataLink...)
				transactionPool.ownerPool.Add(txId, ownerData)

			case *BitmarkTransfer:
				transfer := record.(*BitmarkTransfer)

				// previous record
				previousLink := transfer.Link.Bytes()
				previous, found := transactionPool.ownerPool.Get(previousLink)
				if !found {
					fault.PanicWithError("transaction.SetState", fault.ErrLinkNotFound)
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
				fault.Panic("transaction.SetState - unknown transaction type")
			}

			ok = true
		default:
		}
	case MinedTransaction:
	}

	// should not happen, code is broken - so panic
	if !ok {
		fault.Criticalf("changing state on txid: %#v", link)
		fault.Criticalf("from: '%c'(%d)  to: '%c'(%d)  is forbidden", oldState, oldState, newState, newState)
		fault.Panic("transaction.SetState: invalid state change")
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
		fault.Criticalf("received tx: %x", data)
		fault.Criticalf("local copy:  %x", result)
		fault.Critical("different! => panic")
		fault.Panic("transaction.pool.Exists: database corruption detected")
	}

	// found the record
	return id, true
}
