// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package difficulty

import (
	"fmt"
	"github.com/bitmark-inc/bitmarkd/difficulty/filters"
	"github.com/bitmark-inc/bitmarkd/fault"
	"math"
	"math/big"
	"sync"
)

// the default uint32 value
const DefaultUint32 = 0x1d00ffff

// Type for difficulty
type Difficulty struct {
	sync.RWMutex

	big   big.Int // master value 256 bit integer in pool difficulty form
	pdiff float64 // cache: pool difficulty
	bits  uint32  // cache: bitcoin difficulty

	modifier int            // filter backoff counter
	filter   filters.Filter // filter for difficulty auto-adjust
}

// current difficulty
var Current = &Difficulty{
	filter: filters.NewCamm(1.0, 21, 41),
}

// constOne is for "pdiff" calculation as defined by:
//   https://en.bitcoin.it/wiki/Difficulty#How_is_difficulty_calculated.3F_What_is_the_difference_between_bdiff_and_pdiff.3F
//
// pool difficulty of 1
var constOne = []byte{
	0x00, 0x00, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
}

// number of decimal places
const constScale = 1000000000000

var scale big.Int // 10 times bigger for rounding
var one big.Int   // for reciprocal calculation

// on startup
func init() {
	one.SetBytes(constOne)
	scale.SetUint64(10 * constScale)
	Current.SetBits(0x1d00ffff)
}

// create a difficulty with the default value
func New() *Difficulty {
	d := new(Difficulty)
	return d.SetBits(DefaultUint32)
}

// Get 1/difficulty as normal floating-point value
// this is the Pdiff value
func (difficulty *Difficulty) Pdiff() float64 {
	difficulty.RLock()
	defer difficulty.RUnlock()
	return difficulty.pdiff
}

// Get difficulty as short packed value
func (difficulty *Difficulty) Bits() uint32 {
	difficulty.RLock()
	defer difficulty.RUnlock()
	return difficulty.bits
}

// Get difficulty as the big endian hex encodes short packed value
func (difficulty *Difficulty) String() string {
	difficulty.RLock()
	defer difficulty.RUnlock()
	return fmt.Sprintf("%08x", difficulty.bits)
}

// for the %#v format use 256 bit value
func (difficulty *Difficulty) GoString() string {
	return fmt.Sprintf("%064x", difficulty.BigInt())
}

// convert a uint32 difficulty value to a big.Int
func (difficulty *Difficulty) BigInt() *big.Int {
	difficulty.RLock()
	defer difficulty.RUnlock()
	d := new(big.Int)
	return d.Set(&difficulty.big)
}

// reset difficulty to 1.0
// ensure write locked before calling this
func (difficulty *Difficulty) internalSetToUnity() *Difficulty {
	difficulty.big.Set(&one)
	difficulty.pdiff = 1.0
	difficulty.bits = DefaultUint32
	return difficulty
}

// set from a 32 bit word (bits)
func (difficulty *Difficulty) SetBits(u uint32) *Difficulty {

	// quick setup for default
	if DefaultUint32 == u {
		difficulty.Lock()
		defer difficulty.Unlock()
		return difficulty.internalSetToUnity()
	}

	exponent := 8 * (int(u>>24)&0xff - 3)
	mantissa := int64(u & 0x00ffffff)

	if mantissa > 0x7fffff || mantissa < 0x008000 || exponent < 0 {
		fault.Criticalf("difficulty.SetBits(0x%08x) invalid value", u)
		fault.Panic("difficulty.SetBits: failed")
	}
	d := big.NewInt(mantissa)
	d.Lsh(d, uint(exponent))

	// compute 1/d
	q := new(big.Int)
	r := new(big.Int)
	q.DivMod(&one, d, r)
	r.Mul(r, &scale) // note: big scale == 10 * constScale
	r.Div(r, d)

	result := float64(q.Uint64())
	result += float64((r.Uint64()+5)/10) / constScale

	// modify cache
	difficulty.Lock()
	defer difficulty.Unlock()

	difficulty.big.Set(d)
	difficulty.pdiff = result
	difficulty.bits = u

	return difficulty
}

func (difficulty *Difficulty) SetPdiff(f float64) {
	difficulty.Lock()
	defer difficulty.Unlock()
	difficulty.internalSetPdiff(f)
}

// ensure write locked before calling this
func (difficulty *Difficulty) internalSetPdiff(f float64) float64 {
	if f <= 1.0 {
		difficulty.internalSetToUnity()
		return 1.0
	}
	difficulty.pdiff = f

	intPart := math.Trunc(f)
	fracPart := math.Trunc((f - intPart) * 10 * constScale)

	q := new(big.Int)
	r := new(big.Int)

	q.SetUint64(uint64(intPart))
	r.SetUint64(uint64(fracPart))
	q.Mul(&scale, q)
	q.Add(q, r)

	q.DivMod(&one, q, r) // can get divide by zero error

	q.Mul(&scale, q)
	q.Add(q, r)
	difficulty.big.Set(q)

	buffer := q.Bytes()
	for i, b := range buffer {
		if 0 != 0x80&b {
			e := uint32(len(buffer) - i + 1)
			u := e<<24 | uint32(b)<<8
			if i+1 < len(buffer) {
				u |= uint32(buffer[i+1])
			}
			if i+2 < len(buffer) && 0 != 0x80&buffer[i+2] {
				if 0 == 0x00ff000&(u+1) {
					u += 1
				}
			}
			difficulty.bits = u
			break
		} else if 0 != b {
			e := uint32(len(buffer) - i)
			u := e<<24 | uint32(b)<<16
			if i+1 < len(buffer) {
				u |= uint32(buffer[i+1]) << 8
			}
			if i+2 < len(buffer) {
				u |= uint32(buffer[i+2])
			}
			if i+3 < len(buffer) && 0 != 0x80&buffer[i+3] {
				if 0 == 0x00800000&(u+1) {
					u += 1
				}
			}
			difficulty.bits = u
			break
		}
	}

	return difficulty.pdiff
}

func (difficulty *Difficulty) SetBytes(b []byte) *Difficulty {

	if len(b) < 4 {
		panic("too few bytes")
	}
	u := uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24

	return difficulty.SetBits(u)
}

// adjustment based on error from desired cycle time
// call as difficulty.Adjust(expectedMinutes, actualMinutes)
func (difficulty *Difficulty) Adjust(expectedMinutes float64, actualMinutes float64) float64 {
	difficulty.Lock()
	defer difficulty.Unlock()

	// reset modifier
	difficulty.modifier = 0

	// if k > 1 then difficulty is too low
	k := expectedMinutes / actualMinutes

	newPdiff := k * difficulty.pdiff

	// compute filter
	newPdiff = difficulty.filter.Process(newPdiff)

	// adjust difficulty
	return difficulty.internalSetPdiff(newPdiff)
}

// logarithmic backoff of difficulty
// call each cycle period if no sucessful block was mined
// the modifier value is the number of cycles (without a block being mined)
// and must be rest whenever a new block is mined or accepted from the network
func (difficulty *Difficulty) Backoff() float64 {
	difficulty.Lock()
	defer difficulty.Unlock()

	switch {
	case difficulty.modifier < 1:
		difficulty.modifier = 1
	case difficulty.modifier > 19:
		difficulty.modifier = 19
	default:
		difficulty.modifier += 1
	}

	// return difficulty.internalSetPdiff(difficulty.pdiff * math.Log10(10-0.5*float64(difficulty.modifier)))

	// disble backoff
	return difficulty.pdiff
}
