// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package observer

import (
	"time"

	"github.com/bitmark-inc/logger"

	"github.com/bitmark-inc/bitmarkd/announce/receptor"
	peerlib "github.com/libp2p/go-libp2p-core/peer"
)

const (
	updatetimeEvent = "updatetime"
)

type updatetime struct {
	receptors receptor.Receptor
	log       *logger.L
}

func (u updatetime) Update(str string, args [][]byte) {
	if str == updatetimeEvent {
		id, err := peerlib.IDFromBytes(args[0])
		if err != nil {
			u.log.Warn(err.Error())
		}
		u.receptors.UpdateTime(id, time.Now())
	}
}

func NewUpdatetime(receptors receptor.Receptor, log *logger.L) Observer {
	return &updatetime{receptors: receptors, log: log}
}
