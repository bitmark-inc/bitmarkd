// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package bitcoin

import (
	"github.com/bitmark-inc/bitmarkd/fault"
)

// fetch transaction as raw hex string
func bitcoinGetRawTransactionHex(hash string) (string, error) {
	globalData.Lock()
	defer globalData.Unlock()

	if !globalData.initialised {
		return "", fault.ErrNotInitialised
	}

	arguments := []interface{}{
		hash,
		0,
	}
	var reply string
	err := bitcoinCall("getrawtransaction", arguments, &reply)
	return reply, err
}
