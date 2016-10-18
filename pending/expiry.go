// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package pending

import (
	"time"
)

// the maximum time before a pending transfer is released
// this must be longer that the payment expiry
const (
	timeout = 60 * time.Minute
)

// expiry loop
func (state *expiryData) Run(args interface{}, shutdown <-chan struct{}) {

	log := state.log
	global := args.(*globalDataType)

	globalData.log.Info("starting…")

loop:
	for {
		log.Info("waiting…")
		select {
		case <-shutdown:
			break loop

		case <-time.After(timeout):
			for i := 0; i < shards; i += 1 {
				global.cache[i].Lock()
				for k, item := range global.cache[i].table {
					if time.Since(item.timestamp) > timeout {
						log.Infof("expired: %#v", k)
						delete(global.cache[i].table, k)
					}
				}
				global.cache[i].Unlock()
			}

		}
	}
}
