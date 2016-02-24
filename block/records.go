// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package block

import (
	"encoding/binary"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/bitmark-inc/bitmarkd/fault"
)

// packed records are just a byte slice
type PackedHeader []byte

// constant values
const (
	Version = 2 // simulating block headers compatible with this Bitcoin version
)

// byte sizes for various fields these match Bitcoin header component sizes
const (
	versionSize       = 4          // Block version number
	previousBlockSize = DigestSize // 256-bit hash of the previous block header
	merkleRootSize    = DigestSize // 256-bit hash based on all of the transactions in the block
	timeSize          = 4          // Current timestamp as seconds since 1970-01-01T00:00 UTC
	bitsSize          = 4          // Current target in compact format
	nonceSize         = 4          // 32-bit number (starts at 0)
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
	Version       uint32
	PreviousBlock Digest
	MerkleRoot    Digest
	Time          uint32
	Bits          difficulty.Difficulty
	Nonce         uint32
}

// turn a byte slice into a record
func (record PackedHeader) Unpack(header *Header) error {
	if len(record) != totalBlockSize {
		return fault.ErrInvalidBlockHeader
	}

	header.Version = binary.LittleEndian.Uint32(record[versionOffset:])

	err := DigestFromBytes(&header.PreviousBlock, record[previousBlockOffset:merkleRootOffset])
	if nil != err {
		return err
	}

	err = DigestFromBytes(&header.MerkleRoot, record[merkleRootOffset:timeOffset])
	if nil != err {
		return err
	}

	header.Time = binary.LittleEndian.Uint32(record[timeOffset:])
	header.Bits.SetBytes(record[bitsOffset:])
	header.Nonce = binary.LittleEndian.Uint32(record[nonceOffset:])

	return nil
}

// digest for a packed
func (record PackedHeader) Digest() Digest {
	return NewDigest(record)
}

// turn a record into an array of bytes
//
// the byte ordering matches a Bitcoin header
func (header *Header) Pack() PackedHeader {
	buffer := make([]byte, totalBlockSize)

	binary.LittleEndian.PutUint32(buffer[versionOffset:], header.Version)

	// these are in little endian order so can just copy them
	copy(buffer[previousBlockOffset:], header.PreviousBlock[:])
	copy(buffer[merkleRootOffset:], header.MerkleRoot[:])

	binary.LittleEndian.PutUint32(buffer[timeOffset:], header.Time)
	binary.LittleEndian.PutUint32(buffer[bitsOffset:], header.Bits.Bits())
	binary.LittleEndian.PutUint32(buffer[nonceOffset:], header.Nonce)

	return buffer
}
