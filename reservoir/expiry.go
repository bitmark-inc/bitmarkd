// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package reservoir

import (
	"time"
)

// cleanup cycle time
const (
	timeout = 60 * time.Minute
)

// expiry loop
func (state *expiryData) Run(args interface{}, shutdown <-chan struct{}) {

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
			globalData.Lock()

			for payId, item := range globalData.unverified.entries {
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
			globalData.Unlock()
		}
	}
}
