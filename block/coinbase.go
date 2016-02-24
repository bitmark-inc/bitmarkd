// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package block

import (
	"bytes"
	"encoding/binary"
	"github.com/bitmark-inc/bitmarkd/fault"
	"time"
)

/*

here is the sample from: http://mining.bitcoin.cz/stratum-mining/

coinbase 1: "01000000010000000000000000000000000000000000000000000000000000000000000000ffffffff20020862062f503253482f04b8864e5008"
coinbase 2: "072f736c7573682f000000000100f2052a010000001976a914d23fcdf86f7e756a64a7a9688ef9903327048ed988ac00000000"

this is from block: 25096 [0x6208]   http://blockexplorer.com/testnet/block/000000002076870fe65a2b6eeed84fa892c0db924f1482243a6247d931dcab32

coinbase section from raw block:  020862062f503253482f04b8864e50080800000200000001072f736c7573682f
span between 1 and 2 from sample: 020862062f503253482f04b8864e5008                072f736c7573682f
(shows where nonces are inserted)

Analysis of the sample:

version        = 01000000
input count    = 01
prev input H   = 0000000000000000000000000000000000000000000000000000000000000000
prev input N   = ffffffff
script length  = 20                               === 32 bytes
script         = 02   08 62                       (block number:  25096)
                 06   2f 50  32 53   48 2f        "/P2SH/"
                 04   b8 86  4e 50                (1347323576)
                 08   08 00  00 02   00 00  00 01 (extra nonce 1,2)
                 07   2f 73  6c 75   73 68  2f    "/slush/"
sequence       = 00000000

output count   = 01
BTC            = 00f2052a01000000  === BTC 50.00000000
script length  = 19                === 25 bytes
output script  = 76   OP_DUP
                 a9   OP_HASH160
                 14   d2 3f  cd f8   6f 7e  75 6a   64 a7  a9 68   8e f9  90 33   27 04  8e d9  (20 byte little endian hash value)
                 88   OP_EQUALVERIFY
                 ac   OP_CHECKSIG
sequence       = 00000000

*/

/*

fake coinbase for bitmark:
items:
1. xxxx = extra nonce space
2. yyyy = miner Bitcoin address
3. the "in" encodes block number (minerd compatible value) and nonce
4. each "out" encodes a payment address as: OP_RETURN; byte(currency-length); Currency; address

version        = 01000000
input count    = 01
prev input H   = 0000000000000000000000000000000000000000000000000000000000000000
prev input N   = ffffffff
script length  = mm                               === 31 + bytes
script         = 0n   01 00                       (block number: 1) (n 2..8 depending on how many bytes needed
                 0n   tt tt  tt tt                (timestamp: UTC Unix time in seconds) (t 4..8 depending on how many bytes needed
                 08   xx xx  xx xx   xx xx  xx xx (extra nonce 1,2)
sequence       = 00000000

output count   = nn                === number of addresses

Each out:
---------
BTC            = 0000000000000000  === BTC 0.00000000
script length  = ss                === 1*(OP_RETURN) + 2*(byte-count) + len(currency) + len(address) {must be < 256}
output script  = 6a   OP_RETURN    (marks transaction as invalid)
                 cc   "some currency string"
                 aa   "some address string"
sequence       = 00000000          === 0,1,2,... little endian uint32

Additional Notes:

  1. Possible extension is to create more in or out containing addresses for other currencies (multi currency support)

*/

// some limits
const (
	minimumBlockNumberLength = 2
	maximumBlockNumberLength = 8
	minimumTimestampLength   = 4
	maximumTimestampLength   = 8
	minimumNonceSize         = 4
	maximumNonceSize         = 16

	minimumAddressCount = 1
	maximumAddressCount = 10
)

// bitcoin operations
const (
	OP_RETURN = byte(0x6a)
)

// some offsets into the coinbase
const (
	scriptLengthOffset  = 4 + 1 + DigestSize + 4 // offset of the script size
	minimumScriptLength = 1 + minimumBlockNumberLength + 1 + minimumTimestampLength + 1 + minimumNonceSize
)

// prefix and initial input
var prefixBytes = []byte{
	0x01, 0x00, 0x00, 0x00, // version
	0x01,                                           // input count
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // prev input hash
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // all zero
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // ...
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // ...
	0xff, 0xff, 0xff, 0xff, // prev input N
}

// suffix input sequence
var suffixBytes = []byte{
	0x00, 0x00, 0x00, 0x00, // input sequence
}

// the packed coinbase data
type PackedCoinbase []byte

// type to hold some data unpacked from the coinbase
type CoinbaseData struct {
	BlockNumber uint64
	Timestamp   time.Time
	Addresses   []MinerAddress
}

// create a split coinbase transaction for the miner
// using the current block number
func CurrentCoinbase(timestamp time.Time, nonceSize int, addresses []MinerAddress) (cb1 []byte, cb2 []byte) {
	globalBlock.RLock()
	defer globalBlock.RUnlock()
	//timestamp := time.Now().UTC()
	return NewCoinbase(globalBlock.currentBlockNumber, timestamp, nonceSize, addresses)
}

// create a split coinbase transaction for the miner
func NewCoinbase(blockNumber uint64, timestamp time.Time, nonceSize int, addresses []MinerAddress) (cb1 []byte, cb2 []byte) {

	// nonce will probably be 8 bytes, 4 for bitmarkd and 4 for minerd
	// but allow for double that
	if nonceSize < minimumNonceSize || nonceSize > maximumNonceSize {
		fault.Criticalf("invalid nonce size: %d outside [%d..%d]", nonceSize, minimumNonceSize, maximumNonceSize)
		fault.Panic("invalid nonce size")
	}

	// build coinbase1
	prefixLength := len(prefixBytes)

	// block number (will be 2 .. 8 bytes) (cpuminer currently only recognises 2..4)
	blockNumberLength, blockNumberBytes := byteLength(blockNumber)

	// timestamp (will be 4 .. 8 bytes)
	timestampLength, timestampBytes := byteLength(uint64(timestamp.UTC().Unix()))

	// part one length = prefix length
	//                 + byte(script length)
	//                 + opcode(1:block number) + bytes(block number length)
	//                 + opcode(1:timestamp)    + bytes(timestamp length)
	//                 + opcode(1:nonce)
	// nonce (external)
	// part two length = suffix length
	//                 + outs

	const scriptLengthByte = 1
	const opcodeCount = 3

	// amount of embedded script data (includes nonce op code, but not nonce data)
	scriptLength := opcodeCount + blockNumberLength + timestampLength

	coinbase1 := make([]byte, prefixLength+scriptLengthByte+scriptLength)
	copy(coinbase1, prefixBytes)

	// actual length need to include nonce data
	coinbase1[prefixLength] = byte(scriptLength + nonceSize)

	// embed the block number
	blockNumberOffset := prefixLength + 1
	coinbase1[blockNumberOffset] = byte(blockNumberLength) // push [1..75] bytes op code
	blockNumberOffset += 1
	copy(coinbase1[blockNumberOffset:blockNumberOffset+blockNumberLength], blockNumberBytes)

	// embed the timestamp
	timestampOffset := blockNumberOffset + blockNumberLength
	coinbase1[timestampOffset] = byte(timestampLength) // push [1..75] bytes op code
	timestampOffset += 1
	copy(coinbase1[timestampOffset:timestampOffset+timestampLength], timestampBytes)

	// embed the nonce code
	nonceOffset := timestampOffset + timestampLength
	coinbase1[nonceOffset] = byte(nonceSize) // push [1..75] bytes op code

	// embed addresses
	outs := makeOuts(addresses)
	return coinbase1, append(suffixBytes, outs...)
}

// create a full coinbase transaction for the block
func NewFullCoinbase(blockNumber uint64, timestamp time.Time, nonce []byte, addresses []MinerAddress) (cb PackedCoinbase) {
	nonceSize := len(nonce)
	cb1, cb2 := NewCoinbase(blockNumber, timestamp, nonceSize, addresses)
	return append(append(cb1, nonce...), cb2...)
}

// unpack some data from the coinbase
func (coinbase PackedCoinbase) Unpack(coinbaseData *CoinbaseData) error {

	if len(coinbase) < scriptLengthOffset+minimumScriptLength {
		return fault.ErrInvalidCoinbase
	}

	if !bytes.Equal(coinbase[:len(prefixBytes)], prefixBytes) {
		return fault.ErrInvalidCoinbase
	}

	scriptLength := int(coinbase[scriptLengthOffset])
	script := coinbase[scriptLengthOffset+1 : scriptLengthOffset+1+scriptLength]

	if len(script) != scriptLength {
		return fault.ErrInvalidCoinbase
	}

	// extract block number
	blockNumberLength := int(script[0])
	if blockNumberLength < minimumBlockNumberLength || blockNumberLength > maximumBlockNumberLength {
		return fault.ErrInvalidCoinbase
	}

	n := uint64(0)
	s := uint(0)
	for i := 1; i <= blockNumberLength; i += 1 {
		n |= uint64(script[i]) << s
		s += 8
	}
	coinbaseData.BlockNumber = n

	// extract timestamp
	timestampOffset := blockNumberLength + 1
	timestampLength := int(script[timestampOffset])
	if timestampLength < minimumTimestampLength || timestampLength > maximumTimestampLength {
		return fault.ErrInvalidCoinbase
	}

	t := int64(0)
	s = uint(0)
	for i := 1; i <= timestampLength; i += 1 {
		t |= int64(script[timestampOffset+i]) << s
		s += 8
	}
	coinbaseData.Timestamp = time.Unix(t, 0)

	// determine start of out array
	outsOffset := scriptLengthOffset + 1 + scriptLength + len(suffixBytes)

	outCount := int(coinbase[outsOffset])
	outs := coinbase[outsOffset+1:]
	if outCount > maximumAddressCount {
		outCount = maximumAddressCount
	}

	// extract addresses
	outOffset := 0
	coinbaseData.Addresses = make([]MinerAddress, outCount)
	for i := 0; i < outCount; i += 1 {
		// BTC[8] | count[1] | OP_RETURN[1] | address data | n[4]
		outOffset += 8 // skip BTC value
		outLength := int(outs[outOffset])
		if outLength < 3 {
			return fault.ErrInvalidCoinbase
		}

		if OP_RETURN != outs[outOffset+1] {
			return fault.ErrInvalidCoinbase
		}

		if err := MinerAddressFromBytes(&coinbaseData.Addresses[i], outs[outOffset+2:outOffset+outLength+1]); err != nil {
			return err
		}
		// offset of dext out
		outOffset += outLength + 5
	}

	return nil //////MinerAddressFromBytes(&coinbaseData.Addresses[0], script[addressOffset+1:addressOffset+1+addressLength])
}

// make an array of outs
//
// out
//   nn,                     // output count
// REPEAT:
//   0x00, 0x00, 0x00, 0x00, // BTC 0.00000000 (ls)
//   0x00, 0x00, 0x00, 0x00, // ...            (ms)
//   ss,                     // script length
//   0x6a,                   // OP_RETURN      (marks transaction as invalid)
//   cc    "some currency string"
//   aa    "some address string"
//   0x00, 0x00, 0x00, 0x00, // output sequence
func makeOuts(addresses []MinerAddress) []byte {
	l := len(addresses)

	if l < minimumAddressCount {
		fault.Panic("block: need at least one miner address")
	}

	if l > maximumAddressCount {
		fault.Panic("block: too many miner addresses")
	}

	totalLength := 1 // output count
	for _, a := range addresses {
		totalLength += 8 + 1 + 1 + 2 + 4 // BTC value(8) + scriptLength(1) + OP_RETURN(1) + byteCounts(2) + sequence(4)
		lc := len(a.Currency)
		if lc < minimumCurrencyLength || lc > maximumCurrencyLength {
			fault.Panic("block: invalid currency string")
		}
		la := len(a.Address)
		if lc < minimumAddressLength || lc > maximumAddressLength {
			fault.Panic("block: invalid miner address string")
		}
		totalLength += lc + la
	}

	buffer := make([]byte, totalLength)
	buffer[0] = byte(l) // output count

	i := 1
	sequence := uint32(0)

	for _, a := range addresses {
		binary.LittleEndian.PutUint64(buffer[i:], 0) // BTC = 0.0
		i += 8

		lc := len(a.Currency)
		la := len(a.Address)
		scriptLength := 3 + lc + la
		buffer[i] = byte(scriptLength)
		buffer[i+1] = OP_RETURN
		buffer[i+2] = byte(lc)
		i += 3
		copy(buffer[i:i+lc], a.Currency)
		i += lc
		buffer[i] = byte(la)
		i += 1
		copy(buffer[i:i+la], a.Address)
		i += la
		binary.LittleEndian.PutUint32(buffer[i:], sequence)
		sequence += 1
		i += 4
	}

	return buffer
}

// number of bytes for a 64 bit number
//
// for miner compatibility this will always output a minimum of two bytes
func byteLength(n uint64) (int, []byte) {

	b := []byte{
		byte(n >> 0),
		byte(n >> 8),
		byte(n >> 16),
		byte(n >> 24),
		byte(n >> 32),
		byte(n >> 40),
		byte(n >> 48),
		byte(n >> 56),
	}

	if 0 != n&0xff00000000000000 {
		return 8, b
	}
	if 0 != n&0x00ff000000000000 {
		return 7, b[:7]
	}
	if 0 != n&0x0000ff0000000000 {
		return 6, b[:6]
	}
	if 0 != n&0x000000ff00000000 {
		return 5, b[:5]
	}
	if 0 != n&0x00000000ff000000 {
		return 4, b[:4]
	}
	if 0 != n&0x0000000000ff0000 {
		return 3, b[:3]
	}
	return 2, b[:2]
}
