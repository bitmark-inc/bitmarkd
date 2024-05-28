// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package id_test

import (
	"fmt"
	"testing"

	"github.com/bitmark-inc/bitmarkd/announce/id"
	"github.com/stretchr/testify/assert"
)

func TestCompare(t *testing.T) {
	id1 := id.ID("THIS IS CAPITAL")
	id2 := id.ID("this is normal")
	id3 := id.ID("Same TEXT")
	id4 := id.ID("Same TEXT")

	assert.Equal(t, -1, id1.Compare(id2), "wrong comparison")
	assert.Equal(t, 1, id2.Compare(id1), "wrong comparison")

	//lint:ignore dupArg // really check the argument against itself
	assert.Equal(t, 0, id3.Compare(id3), "wrong comparison")

	assert.Equal(t, 0, id3.Compare(id4), "wrong comparison")
	assert.Equal(t, 0, id4.Compare(id3), "wrong comparison")

}

func TestString(t *testing.T) {
	id1 := id.ID("test string")
	expected := fmt.Sprintf("%x", []byte("test string"))

	assert.Equal(t, expected, id1.String(), "wrong string")
}
