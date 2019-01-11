// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package block

import (
	"github.com/bitmark-inc/bitmarkd/storage"
)

func doRecovery() error {
	return storage.Pool.Blocks.NewFetchCursor().Map(recoverRecord)
}

func recoverRecord(key []byte, value []byte) error {
	return StoreIncoming(value, NoRescanVerified)
}
