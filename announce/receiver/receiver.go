// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package receiver

import (
	"fmt"
	"time"

	peerlib "github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

type Receiver struct {
	ID        peerlib.ID
	Listeners []ma.Multiaddr
	Timestamp time.Time // last seen time
}

// string - conversion from fmt package
func (r Receiver) String() []string {
	allAddress := make([]string, 0)
	for _, listener := range r.Listeners {
		fmt.Println("str: ", listener.String())
		allAddress = append(allAddress, listener.String())
	}
	return allAddress
}
