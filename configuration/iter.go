// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package configuration

import (
	"crypto/rand"
	"encoding/binary"
	"io"
)

const (
	iterSize = 2
)

type Iter [iterSize]byte

func MakeIter() (*Iter, error) {
	iter := new(Iter)
	if _, err := io.ReadFull(rand.Reader, iter[:]); err != nil {
		return iter, err
	}
	intIter := iter.Integer()
	intIter = intIter % 5000
	intIter += 1000
	iter.ConvertIntegerToIter(intIter)
	return iter, nil
}

// convert a binary iter to byte slice
func (iter Iter) Bytes() []byte {
	return iter[:]
}

func (iter Iter) String() string {
	return string(iter.Bytes())
}

func (iter Iter) Integer() int {
	return int(binary.LittleEndian.Uint16(iter.Bytes()))
}

func (iter *Iter) ConvertIntegerToIter(intIter int) {
	binary.LittleEndian.PutUint16(iter[:], uint16(intIter))
}
