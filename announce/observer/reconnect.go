// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package observer

import "github.com/bitmark-inc/bitmarkd/announce/receptor"

const (
	reconnectEvent = "self"
)

type reconnect struct {
	receptors receptor.Receptor
}

func (r reconnect) Update(str string, _ [][]byte) {
	if str == reconnectEvent {
		r.receptors.BalanceTree()
	}
}

func NewReconnect(receptors receptor.Receptor) Observer {
	return &reconnect{receptors: receptors}
}
