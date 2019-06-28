// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package difficulty

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"sync"

	"github.com/bitmark-inc/logger"
)

// the default values
const (
	OneUint64                     uint64  = 0x00ffffffffffffff
	minimumReciprocal             float64 = 1.0
	ExpectedBlockSpacingInSecond          = 2 * 60
	AdjustTimespanInBlocks                = 200
	adjustTimespanInSecond                = ExpectedBlockSpacingInSecond * AdjustTimespanInBlocks
	nextDifficultyRatioUpperbound         = 4
	nextDifficultyRaioLowerbound          = 0.25
	firstBlock                            = 2
	minMutiplyOfTimespanPeriod            = 2
	defaultEmptyBits                      = 8
)

// Difficulty - Type for difficulty
//
// bits is encoded as:
//    8 bit exponent,
//   57 bit mantissa normalised so msb is '1' and omitted
// mantissa is shifted by exponent+8
// examples:
//   the "One" value: 00 ff  ff ff  ff ff  ff ff
//   represents the 256 bit value: 00ff ffff ffff ffff 8000 0000 0000 0000 0000 0000 0000 0000 0000 0000 0000 0000
//   value: 01 ff  ff ff  ff ff  ff ff
//   represents the 256 bit value: 007f ffff ffff ffff c000 0000 0000 0000 0000 0000 0000 0000 0000 0000 0000 0000
type Difficulty struct {
	m          *sync.RWMutex // pointer since MarshallJSON is pass by value
	big        big.Int       // master value 256 bit integer expanded from bits
	reciprocal float64       // cache: floating point reciprocal difficulty
	bits       uint64        // cache: compat difficulty (encoded value)
}

// Current - current difficulty
var Current = New()

// difficulty of 1 as 256 bit big endian value
var constOne = []byte{
	0x00, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
}

var one big.Int        // for reciprocal calculation
var floatOne big.Float // for reciprocal calculation

// on startup
func init() {
	one.SetBytes(constOne)
	floatOne.SetInt(&one)
	Current.SetBits(OneUint64)
}

// New - create a difficulty with the largest possible value
// which is the easiest for the miners and has the fewest leading zeros
func New() *Difficulty {
	d := new(Difficulty)
	return d.internalReset()
}

// Value - difficulty value (floating point, it's Pdiff value)
// This value is a reciprocal, difficulty.value = 1 / difficulty.bits
func (difficulty *Difficulty) Value() float64 {
	difficulty.m.RLock()
	defer difficulty.m.RUnlock()
	return difficulty.reciprocal
}

// Bits - Get difficulty as short packed value
func (difficulty *Difficulty) Bits() uint64 {
	difficulty.m.RLock()
	defer difficulty.m.RUnlock()
	return difficulty.bits
}

// String - Get difficulty as the big endian hex encodes short packed value
func (difficulty *Difficulty) String() string {
	difficulty.m.RLock()
	defer difficulty.m.RUnlock()
	return fmt.Sprintf("%016x", difficulty.bits)
}

// GoString - for the %#v format use 256 bit value
func (difficulty *Difficulty) GoString() string {
	return fmt.Sprintf("%064x", difficulty.BigInt())
}

// BigInt - convert a uint64 difficulty value to a big.Int
func (difficulty *Difficulty) BigInt() *big.Int {
	difficulty.m.RLock()
	defer difficulty.m.RUnlock()
	d := new(big.Int)
	return d.Set(&difficulty.big)
}

// reset difficulty to minimum
// ensure write locked before calling this
func (difficulty *Difficulty) internalReset() *Difficulty {
	if nil == difficulty.m {
		difficulty.m = new(sync.RWMutex)
	}
	difficulty.big.Set(&one)
	difficulty.reciprocal = minimumReciprocal
	difficulty.bits = OneUint64
	return difficulty
}

// SetBits - set from a 64 bit word (bits)
func (difficulty *Difficulty) SetBits(u uint64) *Difficulty {

	// quick setup for default
	if OneUint64 == u {
		difficulty.m.Lock()
		defer difficulty.m.Unlock()
		return difficulty.internalReset()
	}

	exponent := uint(u>>56) & 0xff
	mantissa := u&0x00ffffffffffffff | 0x0100000000000000 // include hidden bit

	// check for exponent overflow
	if exponent >= 0xc0 {
		logger.Criticalf("difficulty.SetBits(0x%16x) invalid value", u)
		logger.Panic("difficulty.SetBits: failed")
	}
	d := big.NewInt(0)
	d.SetUint64(mantissa)
	d.Lsh(d, 256-65-exponent) // account for hidden 56th bit

	// compute 1/d
	denominator := new(big.Float)
	denominator.SetInt(d)
	q := new(big.Float)
	result, _ := q.Quo(&floatOne, denominator).Float64()

	// modify cache
	difficulty.m.Lock()
	defer difficulty.m.Unlock()

	difficulty.big.Set(d)
	difficulty.reciprocal = result
	difficulty.bits = u

	return difficulty
}

// Set - set difficulty value
func (difficulty *Difficulty) Set(f float64) {
	difficulty.m.Lock()
	defer difficulty.m.Unlock()
	difficulty.convertDifficultyIntoReciprocal(f)
}

// ensure write locked before calling this
func (difficulty *Difficulty) convertDifficultyIntoReciprocal(f float64) float64 {
	if f < minimumReciprocal {
		difficulty.internalReset()
		return difficulty.reciprocal
	}
	difficulty.reciprocal = f

	r := new(big.Float)
	r.SetMode(big.ToPositiveInf).SetPrec(256).SetFloat64(f).Quo(&floatOne, r)

	d, _ := r.Int(&difficulty.big)

	// fmt.Printf("f_1: %s\n", floatOne.Text('f', 80))
	// fmt.Printf("rec: %s\n", r.Text('f', 80))
	// fmt.Printf("big: %d\n", d)
	// fmt.Printf("%f\n big: %064x\n", f, d)
	// fmt.Printf("acc: %s\n", accuracy.String())

	buffer := d.Bytes() // no more than 32 bytes (256 bits)

	if len(buffer) > 32 {
		logger.Criticalf("difficulty.convertDifficultyIntoReciprocal(%g) invalid value", f)
		logger.Panic("difficulty.SetBits: failed - needs more than 256 bits")
	}

	// first non-zero byte will not exceed 0x7f as bigints are signed
	// but the above calculation results in an unsigned value
	// need to extract 56 bits with 1st bit as 1  and compute exponent
scan_buffer:
	for i, b := range buffer {
		if 0 != b {
			u := uint64(b) << 56
			if i+1 < len(buffer) {
				u |= uint64(buffer[i+1]) << 48
			}
			if i+2 < len(buffer) {
				u |= uint64(buffer[i+2]) << 40
			}
			if i+3 < len(buffer) {
				u |= uint64(buffer[i+3]) << 32
			}
			if i+4 < len(buffer) {
				u |= uint64(buffer[i+4]) << 24
			}
			if i+5 < len(buffer) {
				u |= uint64(buffer[i+5]) << 16
			}
			if i+6 < len(buffer) {
				u |= uint64(buffer[i+6]) << 8
			}
			if i+7 < len(buffer) {
				u |= uint64(buffer[i+7])
			}

			// compute exponent
			e := uint64(32-len(buffer)+i)*8 - 1

			// normalise
			rounder := 0
			for 0x0100000000000000 != 0xff00000000000000&u {
				if 1 == u&1 {
					rounder += 1
				}
				u >>= 1
				e -= 1
			}

			if rounder > 4 {
				u += 1
			}
			// hide 56th bit and incorporate exponent
			u = u&0x00ffffffffffffff | e<<56
			//fmt.Printf("bits: %016x\n", u)

			difficulty.bits = u
			break scan_buffer
		}
	}

	return difficulty.reciprocal
}

// SetBytes - set the difficulty from little endian bytes
func (difficulty *Difficulty) SetBytes(b []byte) *Difficulty {

	const byteLength = 8
	if len(b) != byteLength {
		logger.Panicf("difficulty.SetBytes: too few bytes expected: %d had: %d", byteLength, len(b))
	}

	u := uint64(b[0]) |
		uint64(b[1])<<8 |
		uint64(b[2])<<16 |
		uint64(b[3])<<24 |
		uint64(b[4])<<32 |
		uint64(b[5])<<40 |
		uint64(b[6])<<48 |
		uint64(b[7])<<56

	return difficulty.SetBits(u)
}

// MarshalText - convert a difficulty to little endian hex for JSON
func (difficulty Difficulty) MarshalText() ([]byte, error) {

	bits := make([]byte, 8)
	binary.LittleEndian.PutUint64(bits, difficulty.bits)

	size := hex.EncodedLen(len(bits))
	buffer := make([]byte, size)
	hex.Encode(buffer, bits)
	return buffer, nil
}

// UnmarshalText - convert a difficulty little endian hex string to difficulty value
func (difficulty *Difficulty) UnmarshalText(s []byte) error {
	buffer := make([]byte, hex.DecodedLen(len(s)))
	_, err := hex.Decode(buffer, s)
	if nil != err {
		return err
	}
	difficulty.internalReset()
	difficulty.SetBytes(buffer)
	return nil
}

// NextDifficultyByPreviousTimespan - next difficulty calculated by previous timespan
func NextDifficultyByPreviousTimespan(prevTimespanSecond uint64, currentDifficulty float64) float64 {
	ratio := adjustRatioByLastTimespan(prevTimespanSecond)

	nextDifficulty := ratio * currentDifficulty

	if nextDifficulty < minimumReciprocal {
		nextDifficulty = minimumReciprocal
	}

	return nextDifficulty
}

func adjustRatioByLastTimespan(actualTimespanSecond uint64) float64 {
	if actualTimespanSecond>>2 >= adjustTimespanInSecond {
		return nextDifficultyRatioUpperbound
	}

	if actualTimespanSecond<<2 <= adjustTimespanInSecond {
		return nextDifficultyRaioLowerbound
	}
	return float64(adjustTimespanInSecond) / float64(actualTimespanSecond)
}

// IsAdjustBlock - is block the one to adjust difficulty
func IsAdjustBlock(height uint64) bool {
	return height%AdjustTimespanInBlocks == 0
}

// PrevTimespanBlockBeginAndEnd - previous begin & end block of difficulty timespan
func PrevTimespanBlockBeginAndEnd(height uint64) (uint64, uint64) {
	if remainder := height % AdjustTimespanInBlocks; remainder != 0 {
		return prevBeginBlockWhenAtBeginOfNextTimespan(height - remainder)
	}
	return prevBeginBlockWhenAtBeginOfNextTimespan(height)
}

func prevBeginBlockWhenAtBeginOfNextTimespan(height uint64) (uint64, uint64) {
	quotient := height / AdjustTimespanInBlocks
	if quotient >= minMutiplyOfTimespanPeriod {
		return height - 1 - AdjustTimespanInBlocks, height - 1
	}

	// below calculation only fits when adjust period in blocks larger than 2
	end := AdjustTimespanInBlocks - 1
	if end <= firstBlock {
		end = AdjustTimespanInBlocks
	}
	return uint64(firstBlock), uint64(end)
}

// Hashrate - calculate hashrate from current difficulty, round value to decimal with 3 digits
func Hashrate() float64 {
	bits := defaultEmptyBits + math.Log2(Current.Value())
	rate := math.Pow(2, bits) / ExpectedBlockSpacingInSecond
	return math.Floor(rate*1000) / 1000
}
