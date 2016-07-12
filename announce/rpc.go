// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package announce

import (
	"encoding/hex"
)

// add an RPC listener
func AddRPC(address string, fingerprint [32]byte) {
	globalData.Lock()
	globalData.rpcs = append(globalData.rpcs, address+" "+hex.EncodeToString(fingerprint[:]))
	globalData.Unlock()
}
