// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package payment

import (
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/bitmark-inc/bitmarkd/mode"
)

// initial difficulty discount scale
// (this will likely remain fixed, increase initialDifficulty instead)
const (
	onef   = 1.00 //        1 item    0%
	tenf   = 0.95 //  2 .. 10 items  -5%
	fiftyf = 0.90 // 11 .. 50 items -10%
	otherf = 0.85 // 51 ..100 items -15%
)

// increase this to make hashing more difficult overall
// (this is then scaled by count*discount)
const (
	//initialDifficulty = 1.0 // 8 leading zero bits
	//initialDifficulty = 256.0 // 16 leading zero bits
	//initialDifficulty = 65536.0 // 24 leading zero bits
	//initialDifficulty = 16777216.0 // 32 leading zero bits
	initialBitmarkDifficulty = 65536.0 // 24 leading zero bits
	initialTestingDifficulty = 256.0   // 16 leading zero bits
)

// produce a scaled difficulty based on the number of items
// in a block to be processed and include a quantity discount
func ScaledDifficulty(count int) *difficulty.Difficulty {

	d := difficulty.New()
	factor := 1.0

	switch {
	case count <= 1:
		factor = onef
	case count <= 10:
		factor = tenf
	case count <= 50:
		factor = fiftyf
	default:
		factor = otherf
	}
	initialDifficulty := initialBitmarkDifficulty
	if mode.IsTesting() {
		initialDifficulty = initialTestingDifficulty
	}
	d.SetReciprocal(float64(count) * initialDifficulty * factor)
	return d
}
