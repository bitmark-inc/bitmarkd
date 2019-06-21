// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockrecord

import (
	"encoding/binary"
	"math/big"
	"time"

	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/storage"
)

// PackedHeader - use fixed size byte array for header to simplify
// validation
type PackedHeader [totalBlockSize]byte

// PackedBlock - packed records are just a byte slice
type PackedBlock []byte

// currently supported block version (used by proofer)
const (
	Version            = 2
	MinimumVersion     = 1
	MinimumBlockNumber = 2 // 1 => genesis block
)

// maximum transactions in a block
// limited by uint16 field
const (
	MinimumTransactions = 2
	MaximumTransactions = 10000
)

// byte sizes for various fields
const (
	VersionSize          = 2                   // Block version number
	TransactionCountSize = 2                   // Count of transactions
	NumberSize           = 8                   // This block's number
	PreviousBlockSize    = blockdigest.Length  // 256-bit Argon2d hash of the previous block header
	MerkleRootSize       = merkle.DigestLength // 256-bit SHA3 hash based on all of the transactions in the block
	TimestampSize        = 8                   // Current timestamp as seconds since 1970-01-01T00:00 UTC
	DifficultySize       = 8                   // Current target difficulty in compact format
	NonceSize            = 8                   // 64-bit number (starts at 0)
)

// offsets of the fields
const (
	versionOffset          = 0
	transactionCountOffset = versionOffset + VersionSize
	numberOffset           = transactionCountOffset + TransactionCountSize
	previousBlockOffset    = numberOffset + NumberSize
	merkleRootOffset       = previousBlockOffset + PreviousBlockSize
	timestampOffset        = merkleRootOffset + MerkleRootSize
	difficultyOffset       = timestampOffset + TimestampSize
	nonceOffset            = difficultyOffset + DifficultySize

	// to set size of header array
	totalBlockSize = nonceOffset + NonceSize // total bytes in the header
)

// Header - the unpacked header structure
type Header struct {
	Version          uint16                 `json:"version"`
	TransactionCount uint16                 `json:"transactionCount"`
	Number           uint64                 `json:"number,string"`
	PreviousBlock    blockdigest.Digest     `json:"previousBlock"`
	MerkleRoot       merkle.Digest          `json:"merkleRoot"`
	Timestamp        uint64                 `json:"timestamp,string"`
	Difficulty       *difficulty.Difficulty `json:"difficulty"`
	Nonce            NonceType              `json:"nonce"`
}

// ExtractHeader - extract a header from the front of a []byte
// if checkHeight non-zero then verify correct block number first
// to reduce hashing load for obviously incorrect blocks
func ExtractHeader(block []byte, checkHeight uint64) (*Header, blockdigest.Digest, []byte, error) {
	if len(block) < totalBlockSize {
		return nil, blockdigest.Digest{}, nil, fault.ErrInvalidBlockHeaderSize
	}
	packedHeader := PackedHeader{}
	copy(packedHeader[:], block[:totalBlockSize])

	header, err := packedHeader.Unpack()
	if nil != err {
		return nil, blockdigest.Digest{}, nil, err
	}

	if checkHeight > 0 && header.Number != checkHeight {
		return nil, blockdigest.Digest{}, nil, fault.ErrHeightOutOfSequence
	}

	var digest blockdigest.Digest
	if storage.Pool.BlockHeaderHash != nil {
		thisBlockNumberKey := make([]byte, 8)
		binary.BigEndian.PutUint64(thisBlockNumberKey, header.Number)
		digestBytes := storage.Pool.BlockHeaderHash.Get(thisBlockNumberKey)
		if err := blockdigest.DigestFromBytes(&digest, digestBytes); err != nil {
			digest = blockdigest.NewDigest(packedHeader[:])
		}
	} else {
		digest = blockdigest.NewDigest(packedHeader[:])
	}

	blockDifficulty := header.Difficulty.BigInt()
	currentDifficulty := difficulty.Current.BigInt()

	n := big.NewInt(10) // range ± N%
	l := big.NewInt(0)
	h := big.NewInt(0)
	h.Quo(currentDifficulty, n)
	l.Sub(currentDifficulty, h) // current - N%
	h.Add(currentDifficulty, h) // current + N%

	if blockDifficulty.Cmp(l) < 0 || blockDifficulty.Cmp(h) > 0 || digest.Cmp(blockDifficulty) > 0 {
		return nil, blockdigest.Digest{}, nil, fault.ErrInvalidBlockHeaderDifficulty
	}

	return header, digest, block[totalBlockSize:], nil

}

// ComputeHeaderHash - return the hash of a block's header
func ComputeHeaderHash(block []byte) (blockdigest.Digest, error) {
	if len(block) < totalBlockSize {
		return blockdigest.Digest{}, fault.ErrInvalidBlockHeaderSize
	}
	packedHeader := PackedHeader{}
	copy(packedHeader[:], block[:totalBlockSize])

	return blockdigest.NewDigest(packedHeader[:]), nil
}

// Unpack - turn a byte slice into a record
func (record PackedHeader) Unpack() (*Header, error) {

	header := &Header{
		Difficulty: difficulty.New(),
	}

	header.Version = binary.LittleEndian.Uint16(record[versionOffset:])
	header.TransactionCount = binary.LittleEndian.Uint16(record[transactionCountOffset:])
	header.Number = binary.LittleEndian.Uint64(record[numberOffset:])

	if 1 == header.Number && 1 == header.TransactionCount && 1 == header.Version {
		// genesis block
	} else {
		// normal block
		if header.Version < MinimumVersion || header.Number < MinimumBlockNumber {
			return nil, fault.ErrInvalidBlockHeaderVersion
		}

		if header.TransactionCount < MinimumTransactions || header.TransactionCount > MaximumTransactions {
			return nil, fault.ErrTransactionCountOutOfRange
		}
	}

	err := blockdigest.DigestFromBytes(&header.PreviousBlock, record[previousBlockOffset:merkleRootOffset])
	if nil != err {
		return nil, err
	}

	err = merkle.DigestFromBytes(&header.MerkleRoot, record[merkleRootOffset:timestampOffset])
	if nil != err {
		return nil, err
	}

	header.Timestamp = binary.LittleEndian.Uint64(record[timestampOffset:difficultyOffset])

	if header.Timestamp > uint64(time.Now().Add(5*time.Minute).Unix()) {
		return nil, fault.ErrInvalidBlockHeaderTimestamp
	}

	header.Difficulty.SetBytes(record[difficultyOffset:nonceOffset])
	header.Nonce = NonceType(binary.LittleEndian.Uint64(record[nonceOffset:]))

	return header, nil
}

// Digest - digest for a packed header
// make sure to truncate bytes to correct length
func (record PackedHeader) Digest() blockdigest.Digest {
	return blockdigest.NewDigest(record[:])
}

// Pack - turn a record into an array of bytes
func (header *Header) Pack() PackedHeader {
	//buffer := make([]byte, TotalBlockSize)
	buffer := PackedHeader{}

	binary.LittleEndian.PutUint16(buffer[versionOffset:], header.Version)
	binary.LittleEndian.PutUint16(buffer[transactionCountOffset:], header.TransactionCount)
	binary.LittleEndian.PutUint64(buffer[numberOffset:], header.Number)

	// these are in little endian order so can just copy them
	copy(buffer[previousBlockOffset:], header.PreviousBlock[:])
	copy(buffer[merkleRootOffset:], header.MerkleRoot[:])

	binary.LittleEndian.PutUint64(buffer[timestampOffset:], header.Timestamp)
	binary.LittleEndian.PutUint64(buffer[difficultyOffset:], header.Difficulty.Bits())
	binary.LittleEndian.PutUint64(buffer[nonceOffset:], uint64(header.Nonce))

	return buffer
}

// FoundationTxId - create the transaction id for a foundation record
// its TxId is sha3-256 . concat blockDigest leBlockNumberUint64
func FoundationTxId(header *Header, digest blockdigest.Digest) merkle.Digest {
	leBlockNumber := make([]byte, 8)
	binary.LittleEndian.PutUint64(leBlockNumber, header.Number)
	return merkle.NewDigest(append(digest[:], leBlockNumber...))
}
