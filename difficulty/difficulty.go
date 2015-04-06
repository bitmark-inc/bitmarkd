// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package difficulty

import (
	"fmt"
	"github.com/bitmark-inc/bitmarkd/fault"
	"math/big"
	"sync"
)

// the default uint32 value
const DefaultUint32 = 0x1d00ffff

// Type for difficulty
type Difficulty struct {
	sync.RWMutex

	big   big.Int // master value
	value float64 // cache
	short uint32  // cache
}

// current difficulty
var Current = &Difficulty{}

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
	Current.SetUint32(0x1d00ffff)
}

// create a difficulty with the default value
func New() *Difficulty {
	d := new(Difficulty)
	return d.SetUint32(DefaultUint32)
}

// Get 1/difficulty as normal floating-point value
func (difficulty *Difficulty) Float64() float64 {
	difficulty.RLock()
	defer difficulty.RUnlock()
	return difficulty.value
}

// Get difficulty as short packed value
func (difficulty *Difficulty) Uint32() uint32 {
	difficulty.RLock()
	defer difficulty.RUnlock()
	return difficulty.short
}

// Get difficulty as the big endian hex encodes short packed value
func (difficulty *Difficulty) String() string {
	difficulty.RLock()
	defer difficulty.RUnlock()
	return fmt.Sprintf("%08x", difficulty.short)
}

// for the %#v format
func (difficulty *Difficulty) GoString() string {
	return difficulty.String()
}

// convert a uint32 difficulty value to a big.Int
func (difficulty *Difficulty) BigInt() *big.Int {
	difficulty.RLock()
	defer difficulty.RUnlock()
	d := new(big.Int)
	return d.Set(&difficulty.big)
}

// set from a 32 bit word
func (difficulty *Difficulty) SetUint32(u uint32) *Difficulty {

	// quick setup for default
	if DefaultUint32 == u {
		difficulty.Lock()
		defer difficulty.Unlock()
		difficulty.big.Set(&one)
		difficulty.value = 1.0
		difficulty.short = u
		return difficulty
	}

	exponent := 8 * (int(u>>24)&0xff - 3)
	mantissa := int64(u & 0x00ffffff)

	if mantissa > 0x7fffff || mantissa < 0x008000 || exponent < 0 {
		fault.Criticalf("difficulty.SetUint32(0x%08x) invalid value", u)
		fault.Panic("difficulty.SetUint32: failed")
	}
	d := big.NewInt(mantissa)
	d.Lsh(d, uint(exponent))

	// compute 1/d
	q := new(big.Int)
	r := new(big.Int)
	q.DivMod(&one, d, r)
	r.Mul(r, &scale)
	r.Div(r, d)

	result := float64(q.Uint64())
	result += float64((r.Uint64()+5)/10) / constScale

	// modify cache
	difficulty.Lock()
	defer difficulty.Unlock()

	difficulty.big.Set(d)
	difficulty.value = result
	difficulty.short = u

	return difficulty
}

func (difficulty *Difficulty) SetBytes(b []byte) *Difficulty {

	if len(b) < 4 {
		panic("too few bytes")
	}
	u := uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24

	return difficulty.SetUint32(u)
}
