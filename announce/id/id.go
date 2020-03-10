// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package id

import (
	"bytes"
	"fmt"
)

type ID []byte

// Compare - public key comparison for AVL interface
func (i ID) Compare(q interface{}) int {
	return bytes.Compare(i, q.(ID))
}

// String - public key string convert for AVL interface
func (i ID) String() string {
	return fmt.Sprintf("%x", []byte(i))
}
