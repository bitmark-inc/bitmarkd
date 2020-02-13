// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package observer

import (
	"github.com/bitmark-inc/bitmarkd/announce/receptor"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/logger"
	"github.com/gogo/protobuf/proto"
	p2pPeer "github.com/libp2p/go-libp2p-core/peer"
	"github.com/prometheus/common/log"
)

const (
	selfEvent = "self"
)

type self struct {
	receptors receptor.Receptor
	log       *logger.L
}

func (s self) Update(str string, args [][]byte) {
	if str == selfEvent {
		id, err := p2pPeer.IDFromBytes(args[0])
		if err != nil {
			log.Warn(err.Error())
		}

		var listeners receptor.Addrs
		_ = proto.Unmarshal(args[1], &listeners)
		addrs := util.GetMultiAddrsFromBytes(listeners.Address)
		if len(addrs) == 0 {
			log.Warn("No valid listener address")
		}
		log.Infof("-><-  request self announce data add to tree: %v  listener: %s", id, receptor.AddrToString(args[1]))

		err = s.receptors.SetSelf(id, addrs)
		if nil != err {
			log.Errorf("announcer set with error: %s", err)
		}
	}
}

func NewSelf(receptors receptor.Receptor, log *logger.L) Observer {
	return &self{receptors: receptors, log: log}
}
