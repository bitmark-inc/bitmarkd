// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package peer

import (
	"github.com/bitmark-inc/bitmarkd/zmqutil"
)

// FetchConnectors - obtain a list of all connector clients
func FetchConnectors() []*zmqutil.Connected {

	globalData.RLock()

	result := make([]*zmqutil.Connected, 0, len(globalData.connectorClients))

	for _, c := range globalData.connectorClients {
		if c != nil {
			connect := c.ConnectedTo()
			if connect != nil {
				result = append(result, connect)
			}
		}
	}

	globalData.RUnlock()

	return result
}
