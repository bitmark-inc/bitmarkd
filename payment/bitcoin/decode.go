// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package bitcoin

import (
	"github.com/bitmark-inc/bitmarkd/fault"
)

// transaction decode
func bitcoinDecodeRawTransaction(hex string, reply *bitcoinTransaction) error {
	globalData.Lock()
	defer globalData.Unlock()

	if !globalData.initialised {
		return fault.ErrNotInitialised
	}

	arguments := []interface{}{
		hex,
	}
	return bitcoinCall("decoderawtransaction", arguments, reply)
}
