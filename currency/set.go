// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package currency

// currency set
type Set struct {
	count int
	bits  uint64
}

func MakeSet(currencies ...Currency) Set {
	s := Set{}
	for _, c := range currencies {
		s.Add(c)
	}
	return s
}

// returns number of currencies in the set
func (set *Set) Count() int {
	return set.count
}

// returns true if already present
func (set *Set) Add(c Currency) bool {
	n := uint64(1) << c
	if uint64(0) != n&set.bits {
		return true
	}
	set.count += 1
	set.bits |= n
	return false
}
