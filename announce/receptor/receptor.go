// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package receptor

import (
	"fmt"
	"time"

	peerlib "github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

type Receptor struct {
	ID        peerlib.ID
	Listeners []ma.Multiaddr
	Timestamp time.Time // last seen time
}

// string - conversion from fmt package
func (r Receptor) String() []string {
	allAddress := make([]string, 0)
	for _, listener := range r.Listeners {
		fmt.Println("str: ", listener.String())
		allAddress = append(allAddress, listener.String())
	}
	return allAddress
}
