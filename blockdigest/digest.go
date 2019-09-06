// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockdigest

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/go-argon2"
	"github.com/bitmark-inc/logger"
)

// Length - number of bytes in the digest
const Length = 32

// internal hashing parameters
const (
	digestMode        = argon2.ModeArgon2d
	digestMemory      = 1 << 17 // 128 MiB
	digestParallelism = 1
	digestIterations  = 4
	digestVersion     = argon2.Version13
)

// Digest - type for a digest
// stored as little endian byte array
// represented as big endian hex value for print
// represented as little endian hex text for JSON encoding
type Digest [Length]byte

// NewDigest - create a digest from a byte slice
func NewDigest(record []byte) Digest {

	context := &argon2.Context{
		Iterations:  digestIterations,
		Memory:      digestMemory,
		Parallelism: digestParallelism,
		HashLen:     Length,
		Mode:        digestMode,
		Version:     digestVersion,
	}

	hash, err := argon2.Hash(context, record, record)
	logger.PanicIfError("block.NewDigest", err)

	var digest Digest
	copy(digest[:], hash)
	return digest
}

// Cmp - convert the hash to its equivalent big.Int
func (digest Digest) Cmp(difficulty *big.Int) int {
	bigEndian := reversed(digest)
	result := new(big.Int)
	return result.SetBytes(bigEndian[:]).Cmp(difficulty)
}

//IsEmpty - is digest empty
func (d Digest) IsEmpty() bool {
	return d == (Digest{})
}

// IsValidByDifficulty - is digest valid by difficulty
func (d Digest) IsValidByDifficulty(diff *difficulty.Difficulty) bool {
	reversedDigest := reversed(d)
	bigEndian := new(big.Int)
	bigEndian.SetBytes(reversedDigest[:])
	return bigEndian.Cmp(diff.BigInt()) <= 0
}

func (d Digest) SmallerDigestThan(target Digest) bool {
	for i := len(d) - 1; i >= 0; i-- {
		if d[i] < target[i] {
			return true
		}
	}
	return false
}

// internal function to return a reversed byte order copy of a digest
func reversed(d Digest) []byte {
	result := make([]byte, Length)
	for i := 0; i < Length; i += 1 {
		result[i] = d[Length-1-i]
	}
	return result
}

// String - convert a binary digest to hex string for use by the fmt package (for %s)
//
// the stored version is in little endian, but the output string is big endian
func (digest Digest) String() string {
	return hex.EncodeToString(reversed(digest))
}

// GoString - convert a binary digest to big endian hex string for use by the fmt package (for %#v)
func (digest Digest) GoString() string {
	return "<Argon2d:" + hex.EncodeToString(reversed(digest)) + ">"
}

// Scan - convert a big endian hex representation to a digest for use by the format package scan routines
func (digest *Digest) Scan(state fmt.ScanState, verb rune) error {
	token, err := state.Token(true, func(c rune) bool {
		if c >= '0' && c <= '9' {
			return true
		}
		if c >= 'A' && c <= 'F' {
			return true
		}
		if c >= 'a' && c <= 'f' {
			return true
		}
		return false
	})
	if nil != err {
		return err
	}
	buffer := make([]byte, hex.DecodedLen(len(token)))
	byteCount, err := hex.Decode(buffer, token)
	if nil != err {
		return err
	}

	for i, v := range buffer[:byteCount] {
		digest[Length-1-i] = v
	}
	return nil
}

// MarshalText - convert digest to little endian hex text
func (digest Digest) MarshalText() ([]byte, error) {
	size := hex.EncodedLen(len(digest))
	buffer := make([]byte, size)
	hex.Encode(buffer, digest[:])
	return buffer, nil
}

// UnmarshalText - convert little endian hex text into a digest
func (digest *Digest) UnmarshalText(s []byte) error {
	buffer := make([]byte, hex.DecodedLen(len(s)))
	byteCount, err := hex.Decode(buffer, s)
	if nil != err {
		return err
	}
	for i, v := range buffer[:byteCount] {
		digest[i] = v
	}
	return nil
}

// DigestFromBytes - convert and validate little endian binary byte slice to a digest
func DigestFromBytes(digest *Digest, buffer []byte) error {
	if Length != len(buffer) {
		return fault.ErrNotLink
	}
	for i := 0; i < Length; i += 1 {
		digest[i] = buffer[i]
	}
	return nil
}
