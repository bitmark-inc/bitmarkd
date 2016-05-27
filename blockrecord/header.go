// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockrecord

import (
	"encoding/binary"
	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
)

// packed records are just a byte slice
type PackedHeader []byte

// block version
const (
	Version = 3
)

// byte sizes for various fields these match Bitcoin header component sizes
const (
	versionSize       = 4                   // Block version number
	previousBlockSize = blockdigest.Length  // 256-bit Argon2d hash of the previous block header
	merkleRootSize    = merkle.DigestLength // 256-bit SHA3 hash based on all of the transactions in the block
	timeSize          = 8                   // Current timestamp as seconds since 1970-01-01T00:00 UTC
	bitsSize          = 8                   // Current target in compact format
	nonceSize         = 8                   // 64-bit number (starts at 0)
)

// offsets of the fields
const (
	versionOffset       = 0
	previousBlockOffset = versionOffset + versionSize
	merkleRootOffset    = previousBlockOffset + previousBlockSize
	timeOffset          = merkleRootOffset + merkleRootSize
	bitsOffset          = timeOffset + timeSize
	nonceOffset         = bitsOffset + bitsSize

	totalBlockSize = nonceOffset + nonceSize // total bytes in the header
)

// the unpacked header structure
// the types here must match Bitcoin header types
type Header struct {
	Version       uint64                 `json:"version"`
	PreviousBlock blockdigest.Digest     `json:"previous_block"`
	MerkleRoot    merkle.Digest          `json:"merkle_root"`
	Time          uint64                 `json:"time,string"`
	Bits          *difficulty.Difficulty `json:"bits"`
	Nonce         NonceType              `json:"nonce"`
}

// turn a byte slice into a record
func (record PackedHeader) Unpack(header *Header) error {
	if len(record) != totalBlockSize {
		return fault.ErrInvalidBlockHeader
	}

	header.Version = binary.LittleEndian.Uint64(record[versionOffset:])

	err := blockdigest.DigestFromBytes(&header.PreviousBlock, record[previousBlockOffset:merkleRootOffset])
	if nil != err {
		return err
	}

	err = merkle.DigestFromBytes(&header.MerkleRoot, record[merkleRootOffset:timeOffset])
	if nil != err {
		return err
	}

	header.Time = binary.LittleEndian.Uint64(record[timeOffset:])
	header.Bits.SetBytes(record[bitsOffset:])
	header.Nonce = NonceType(binary.LittleEndian.Uint64(record[nonceOffset:]))

	return nil
}

// digest for a packed
func (record PackedHeader) Digest() blockdigest.Digest {
	return blockdigest.NewDigest(record)
}

// turn a record into an array of bytes
func (header *Header) Pack() PackedHeader {
	buffer := make([]byte, totalBlockSize)

	binary.LittleEndian.PutUint64(buffer[versionOffset:], header.Version)

	// these are in little endian order so can just copy them
	copy(buffer[previousBlockOffset:], header.PreviousBlock[:])
	copy(buffer[merkleRootOffset:], header.MerkleRoot[:])

	binary.LittleEndian.PutUint64(buffer[timeOffset:], header.Time)
	binary.LittleEndian.PutUint64(buffer[bitsOffset:], header.Bits.Bits())
	binary.LittleEndian.PutUint64(buffer[nonceOffset:], uint64(header.Nonce))

	return buffer
}
