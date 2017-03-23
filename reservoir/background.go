// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package reservoir

import (
	"github.com/bitmark-inc/bitmarkd/storage"
	"time"
)

// cycle time
const (
	timeout = 60 * time.Second
)

// background process loop
func (state *verifierData) Run(args interface{}, shutdown <-chan struct{}) {

	log := state.log
	globalData := args.(*globalDataType)

	log.Info("starting…")

loop:
	for {
		log.Info("waiting…")
		select {
		case <-shutdown:
			break loop

		case <-time.After(timeout):
			state.process(globalData)
		}
	}
}

func (state *verifierData) process(globaldata *globalDataType) {
	log := state.log

	globalData.Lock()
	defer globalData.Unlock()

	for payId, item := range globalData.unverified.entries {
		record := storage.Pool.Payment.Get(payId[:])
		if nil != record {
			setVerified(payId)
			continue
		}

		if time.Since(item.expires) > 0 {
			log.Infof("expired: %#v", payId)

			for _, txId := range item.txIds {
				delete(globalData.unverified.index, txId)
			}

			for _, link := range item.links {
				delete(globalData.pendingTransfer, link)
			}

			delete(globalData.unverified.entries, payId)
		}
	}
}
