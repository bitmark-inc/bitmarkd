// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package broadcast

import (
	"encoding/binary"
	"sync"
	"time"

	"github.com/bitmark-inc/bitmarkd/announce/receptor"
	"github.com/bitmark-inc/bitmarkd/announce/rpc"
	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/logger"
)

type broadcast struct {
	sync.RWMutex
	log                *logger.L
	receptors          receptor.Receptor
	rpcs               rpc.RPC
	initialiseInterval time.Duration
	pollingInterval    time.Duration
}

// Run - background process interface
func (b *broadcast) Run(arg interface{}, shutdown <-chan struct{}) {
	//log := b.log		// ***** FIX THIS: panics in test
	//log.Info("starting…")   // ***** FIX THIS: panics in test

	queue := arg.(<-chan messagebus.Message)

	delay := time.After(b.initialiseInterval)
loop:
	for {
		//log.Debug("waiting…")   // ***** FIX THIS: panics in test
		select {
		case <-shutdown:
			break loop
		case item := <-queue:
			//log.Infof("received control: %s  parameters: %x", item.Command, item.Parameters)   // ***** FIX THIS: panics in test
			switch item.Command {
			case "reconnect":
				b.receptors.ReBalance()
			case "updatetime":
				b.receptors.UpdateTime(item.Parameters[0], time.Now())
			default:
			}

		case <-delay:
			delay = time.After(b.pollingInterval)
			b.process()
		}
	}
}

// process announcement and return response to client
func (b *broadcast) process() {
	log := b.log
	log.Debug("process starting…")

	b.Lock()
	defer b.Unlock()

	// get a big endian timestamp
	timestamp := make([]byte, 8)
	binary.BigEndian.PutUint64(timestamp, uint64(time.Now().Unix()))

	// announce this nodes IP and ports to other peers
	if b.rpcs.IsInitialised() {
		fin := b.rpcs.ID()
		log.Debugf("send rpc: %x", fin)
		messagebus.Bus.Broadcast.Send("rpc", fin[:], b.rpcs.Self(), timestamp)
	}

	if b.receptors.IsInitialised() {
		log.Debugf("send peer: %x", b.receptors.ID())
		messagebus.Bus.Broadcast.Send("peer", b.receptors.ID(), b.receptors.SelfListener(), timestamp)
	}

	b.rpcs.Expire()
	b.receptors.Expire()

	if b.receptors.IsChanged() {
		b.receptors.ReBalance()
		b.receptors.Change(false)
	}
}

// New - return interface for background processing
func New(log *logger.L, receptors receptor.Receptor, rpcs rpc.RPC, initialiseInterval, pollingInterval time.Duration) background.Process {
	return &broadcast{
		log:                log,
		receptors:          receptors,
		rpcs:               rpcs,
		initialiseInterval: initialiseInterval,
		pollingInterval:    pollingInterval,
	}
}
