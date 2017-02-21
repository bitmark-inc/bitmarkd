// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockring

import (
	"hash/crc64"
)

// create the CRC64 table
var table = crc64.MakeTable(crc64.ECMA)

// crc a block digest
func CRC(height uint64, packed []byte) uint64 {
	return crc64.Update(height, table, packed)
}
