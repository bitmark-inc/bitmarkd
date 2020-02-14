// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package observer

import (
	"encoding/binary"
	"fmt"

	"github.com/bitmark-inc/logger"
	p2pPeer "github.com/libp2p/go-libp2p-core/peer"

	"github.com/bitmark-inc/bitmarkd/announce/receptor"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/gogo/protobuf/proto"
)

const (
	addpeerEvent = "addpeer"
)

type addpeer struct {
	receptors receptor.Receptor
	log       *logger.L
}

func (a addpeer) Update(str string, args [][]byte) {
	if str == addpeerEvent {
		id, err := p2pPeer.IDFromBytes(args[0])
		if err != nil {
			a.log.Warn(err.Error())
			return
		}

		var listeners receptor.Addrs
		err = proto.Unmarshal(args[1], &listeners)
		if err != nil {
			util.LogError(a.log, util.CoRed, fmt.Sprintf("addpeer: Unmarshal Address Error:%v", err))
			return
		}

		addrs := util.GetMultiAddrsFromBytes(listeners.Address)
		if len(addrs) == 0 {
			util.LogError(a.log, util.CoRed, "No valid listener address: addrs is empty")
			return
		}

		if len(args[2]) != 8 {
			util.LogError(a.log, util.CoRed, "Invalid timestamp")
			return
		}
		timestamp := binary.BigEndian.Uint64(args[2])
		a.receptors.Add(id, addrs, timestamp)
		util.LogDebug(a.log, util.CoYellow, fmt.Sprintf("-><- addpeer : %s  listener: %s  Timestamp: %d", id.String(), receptor.AddrToString(args[1]), timestamp))
	}
}

func NewAddpeer(receptors receptor.Receptor, log *logger.L) Observer {
	return &addpeer{receptors: receptors, log: log}
}
