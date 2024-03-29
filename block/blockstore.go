// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package block

import (
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/logger"
)

type blockstore struct {
	log *logger.L
}

// initialise the broadcaster
func (blk *blockstore) initialise() error {

	log := logger.New("blockstore")
	blk.log = log

	log.Info("initialising…")

	return nil
}

// wait for new blocks
func (blk *blockstore) Run(args interface{}, shutdown <-chan struct{}) {

	log := blk.log

	log.Info("starting…")

	queue := messagebus.Bus.Blockstore.Chan()

loop:
	for {
		log.Debug("waiting…")
		select {
		case <-shutdown:
			break loop
		case item := <-queue:
			log.Infof("received: %s  data: %x", item.Command, item.Parameters)
			blk.process(&item)
		}
	}
	messagebus.Bus.Blockstore.Release()
}

// process the received block
func (blk *blockstore) process(item *messagebus.Message) {

	log := blk.log

	if len(item.Parameters) == 1 {
		packedBlock := item.Parameters[0]
		err := StoreIncoming(packedBlock, nil, RescanVerified)
		if err == nil {
			// broadcast this packedBlock to peers if the block was valid
			messagebus.Bus.Broadcast.Send("block", packedBlock)
		} else {
			log.Debugf("store block: %x  error: %s", packedBlock, err)
		}
	}
}
