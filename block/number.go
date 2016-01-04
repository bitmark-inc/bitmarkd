// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package block

import (
	"bytes"
	"encoding/binary"
	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/pool"
	"github.com/bitmark-inc/logger"
	"sync"
	"time"
)

// constants
const (
	ExpectedMinutes = 10 // number of minutes to mine a block

	int16Size  = 2 // counts are encode little endian int16
	uint64Size = 8 // for block number key
)

// type to hold the unpacked block
type Block struct {
	Digest    Digest
	Number    uint64
	Timestamp time.Time // the 64 bit timestamp from coinbase, _NOT_ block header ntime
	Addresses []MinerAddress
	Header    Header
	Coinbase  []byte
	TxIds     []Digest
}

// for packed block
type Packed []byte // for general distribution
type Mined []byte  // tags an already mined block so background can ignore it

// global lock for validating block

var globalBlock struct {
	sync.RWMutex

	// set once during Initialise - all routines must error if this is false
	initialised bool

	// channel for logging
	log *logger.L

	// this is the block number of the next block to be mined
	currentBlockNumber uint64

	// link to the previous block
	previousBlock Digest

	// retain the timestamp
	previousTimestamp time.Time

	// stored block data
	blockData *pool.Pool

	// for background processes
	background *background.T
}

// list of background processes to start
var processes = background.Processes{
	decay,
}

// initialise the block numbering system
func Initialise() {

	// ensure single access
	globalBlock.Lock()
	defer globalBlock.Unlock()

	globalBlock.log = logger.New("block")
	globalBlock.log.Info("starting…")

	globalBlock.currentBlockNumber = 0

	globalBlock.blockData = pool.New(pool.BlockData)

	if mode.IsTesting() {
		globalBlock.previousBlock = TestGenesisDigest
	} else {
		globalBlock.previousBlock = LiveGenesisDigest
	}
	globalBlock.currentBlockNumber = GenesisBlockNumber + 1

	globalBlock.initialised = true

	// start background processes
	globalBlock.log.Info("start background")
	globalBlock.background = background.Start(processes, globalBlock.log)

	// determine the highest block on store
	last, found := globalBlock.blockData.LastElement()
	if !found {
		return
	}

	// recover block number
	bn := binary.BigEndian.Uint64(last.Key)
	globalBlock.log.Infof("Highest block on file: %d\n", bn)

	// recover previous block digest
	var blk Block
	err := Packed(last.Value).Unpack(&blk)
	if nil == err {
		globalBlock.currentBlockNumber = bn + 1
		globalBlock.previousBlock = blk.Digest
		globalBlock.previousTimestamp = blk.Timestamp
		return
	}

	// ***** FIX THIS: loop back to see if a lower block is ok *****

	fault.Criticalf("block data corrupted: error: %v\n", err)
	fault.Panic("block data corrupted")
}

// finalise - flush unsaved data
func Finalise() {
	globalBlock.log.Info("shutting down…")

	background.Stop(globalBlock.background)

	globalBlock.blockData.Flush()
}

// access to previous link
func PreviousLink() Digest {
	globalBlock.Lock()
	defer globalBlock.Unlock()
	return globalBlock.previousBlock
}

// access to the block number being mined
func Number() uint64 {
	globalBlock.Lock()
	defer globalBlock.Unlock()
	return globalBlock.currentBlockNumber
}

// create the combined block assuming current state
//
// binary format:
// 1. packed header
// 2. coinbase
//    a. length (little endian int16)
//    b. packed coinbase
// 3. transaction count + 1 (to account for coinbase digest) (little endian int16)
//    a. coinbase digest
//    b. list of txids (digests)
// 4. full merkle tree computed from 3
// validate submitted data, and save the block
func MinerCheckIn(timestamp time.Time, ntime uint32, nonce uint32, extraNonce []byte, addresses []MinerAddress, ids []Digest) (Digest, Packed, bool) {

	// ensure single access
	globalBlock.Lock()
	defer globalBlock.Unlock()

	digest, blk, ok := Pack(globalBlock.currentBlockNumber, timestamp, difficulty.Current, ntime, nonce, extraNonce, addresses, ids)
	if !ok {
		return digest, blk, false
	}

	// store
	blk.internalSave(globalBlock.currentBlockNumber, &digest, timestamp)

	return digest, blk, true
}

// return the Genesis Block
func GenesisBlock() Packed {
	if mode.IsTesting() {
		return TestGenesisBlock
	} else {
		return LiveGenesisBlock
	}
}

// fetch a stored block
func Get(number uint64) (Packed, bool) {

	if number < 1 {
		return nil, false
	}

	// genesis block
	if GenesisBlockNumber == number {
		return GenesisBlock(), true
	}

	blockKey := make([]byte, uint64Size)
	binary.BigEndian.PutUint64(blockKey, number)

	data, found := globalBlock.blockData.Get(blockKey)
	if !found {
		return nil, false
	}
	return Packed(data), true
}

// store a block
// requires number to save an unpack step which was probably already done
// or the number was known from the pack step
func (blk Packed) Save(number uint64, digest *Digest, timestamp time.Time) {
	globalBlock.Lock()
	defer globalBlock.Unlock()
	blk.internalSave(number, digest, timestamp)
}

// this does not lock, so use only when locked
func (blk Packed) internalSave(number uint64, digest *Digest, timestamp time.Time) {

	blockKey := make([]byte, uint64Size)
	binary.BigEndian.PutUint64(blockKey, number)

	globalBlock.log.Infof("storing block %d", number)
	globalBlock.blockData.Add(blockKey, blk)

	// update current block number/digest
	if number >= globalBlock.currentBlockNumber {
		globalBlock.currentBlockNumber = number + 1
		globalBlock.previousBlock = *digest

		// compute decimal minutes taken to mine the block
		// actualMinutes := timestamp.Sub(globalBlock.previousTimestamp).Minutes()
		actualMinutes := timestamp.Sub(globalBlock.previousTimestamp).Minutes()

		globalBlock.log.Debugf("adjust difficulty previous timestamp: %v", globalBlock.previousTimestamp)
		globalBlock.log.Debugf("adjust difficulty current  timestamp: %v", timestamp)
		globalBlock.log.Debugf("adjust difficulty expected: %d min  actual: %10.4f min", ExpectedMinutes, actualMinutes)

		// adjust difficulty
		d := difficulty.Current.Adjust(ExpectedMinutes, actualMinutes)
		globalBlock.log.Debugf("adjust difficulty to: %10.4f", d)


		// save latest timestamp
		globalBlock.previousTimestamp = timestamp
	}
}

// create a packed block from various pieces
func Pack(blockNumber uint64, timestamp time.Time, difficulty *difficulty.Difficulty, ntime uint32, nonce uint32, extraNonce []byte, addresses []MinerAddress, ids []Digest) (Digest, Packed, bool) {

	// ensure transactions fit in int16
	transactionCount := len(ids) + 1
	if transactionCount > 32767 {
		fault.Panic("block.Pack - transaction count exceeds 32767")
	}

	// coinbase
	coinbase := NewFullCoinbase(blockNumber, timestamp, extraNonce, addresses)
	cDigest := NewDigest(coinbase)
	coinbaseLength := len(coinbase)

	// ensure coinbase length will fit in int16
	if coinbaseLength > 32767 {
		fault.Panic("block.Check - coinbase exceeds 32767 bytes")
	}

	// merkle tree
	tree := FullMerkleTree(cDigest, ids)

	// block header
	h := Header{
		Version:       Version,
		PreviousBlock: globalBlock.previousBlock,
		MerkleRoot:    tree[len(tree)-1],
		Time:          ntime,
		Bits:          *difficulty,
		Nonce:         nonce,
	}

	header := h.Pack()
	hDigest := header.Digest()

	// check difficulty
	if hDigest.Cmp(difficulty.BigInt()) > 0 {
		return hDigest, nil, false
	}

	// compute block size
	blockSize := len(header) + 2*int16Size + coinbaseLength + len(tree)*DigestSize

	blk := make([]byte, 0, blockSize)
	blk = append(blk, header...)
	blk = append(blk, byte(coinbaseLength&0xff))
	blk = append(blk, byte(coinbaseLength>>8))
	blk = append(blk, coinbase...)
	blk = append(blk, byte(transactionCount&0xff))
	blk = append(blk, byte(transactionCount>>8))

	buffer := new(bytes.Buffer)
	err := binary.Write(buffer, binary.LittleEndian, tree)
	fault.PanicIfError("block.Check - writing merkle", err)

	blk = append(blk, buffer.Bytes()...)

	if len(blk) != blockSize {
		fault.Criticalf("block.Check - block size mismatch: actual: %d, expected: %d", len(blk), blockSize)
		fault.Panic("block.Check - block size mismatch")
	}

	return hDigest, blk, true
}

func (pack Packed) Unpack(blk *Block) error {

	// see if header + coinbase length
	if len(pack) < totalBlockSize+int16Size {
		return fault.ErrInvalidBlock
	}

	// compute coinbase length and verify
	coinbaseLength := int(pack[totalBlockSize]) + int(pack[totalBlockSize+1])<<8
	if coinbaseLength < 1 || coinbaseLength > 32767 {
		return fault.ErrInvalidBlock
	}

	// extent of coinbase
	cbStart := totalBlockSize + int16Size
	cbFinish := cbStart + coinbaseLength

	// check long enough for coinbase + transaction count
	if len(pack) < cbFinish+int16Size {
		return fault.ErrInvalidBlock
	}

	// extract coinbase
	coinbase := PackedCoinbase(pack[cbStart:cbFinish])

	// extract useful data from coinbase
	var cb CoinbaseData
	err := coinbase.Unpack(&cb)
	if nil != err {
		return err
	}

	// compute transaction count and verify
	txCount := int(pack[cbFinish]) + int(pack[cbFinish+1])<<8
	if txCount < 1 || txCount > 32767 {
		return fault.ErrInvalidBlock
	}

	// extract merkle tree bytes
	// check proper multiple of digest size
	merkleBytes := pack[cbFinish+int16Size:]
	digestCount := len(merkleBytes) / DigestSize
	if len(merkleBytes) != digestCount*DigestSize {
		return fault.ErrInvalidBlock
	}

	// extract transaction ids
	var cbId Digest
	err = DigestFromBytes(&cbId, merkleBytes[:DigestSize])
	if nil != err {
		return err
	}

	// read transaction ids
	txIds := make([]Digest, txCount-1)
	offset := 0
	for i := 0; i < txCount-1; i += 1 {
		offset += DigestSize
		err = DigestFromBytes(&txIds[i], merkleBytes[offset:offset+DigestSize])
		if nil != err {
			return err
		}
	}

	// verify coinbase digest matches
	cDigest := NewDigest(coinbase)
	if cDigest != cbId {
		return fault.ErrInvalidBlock
	}

	// verify merkle tree
	tree := FullMerkleTree(cDigest, txIds)
	treeBuffer := new(bytes.Buffer)
	err = binary.Write(treeBuffer, binary.LittleEndian, tree)
	fault.PanicIfError("block.Check - writing merkle", err)

	if !bytes.Equal(treeBuffer.Bytes(), merkleBytes) {
		return fault.ErrInvalidBlock
	}

	// verify header
	blk.Digest = NewDigest(pack[:totalBlockSize])
	err = PackedHeader(pack[:totalBlockSize]).Unpack(&blk.Header)
	if nil != err {
		return err
	}

	if blk.Header.MerkleRoot != tree[len(tree)-1] {
		return fault.ErrInvalidBlock
	}

	if blk.Digest.Cmp(blk.Header.Bits.BigInt()) > 0 {
		return fault.ErrInvalidBlock
	}

	blk.Number = cb.BlockNumber
	blk.Timestamp = cb.Timestamp.UTC()
	blk.Addresses = cb.Addresses

	blk.TxIds = txIds

	return nil
}

// difficulty decay background
// assemble records for mining
func decay(args interface{}, shutdown <-chan bool, finished chan<- bool) {

	log := args.(*logger.L)
	log.Info("decay: starting…")

loop:
	for {
		select {
		case <-shutdown:
			break loop

		case <-time.After(ExpectedMinutes * time.Minute):
			if mode.Is(mode.Normal) {
				d := difficulty.Current.Decay()
				log.Infof("decay difficulty to: %10.4f", d)
			}
		}
	}

	log.Info("decay: shutting down…")
	close(finished)
}
