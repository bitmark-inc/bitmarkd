// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package asset

import (
	"container/list"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"time"
)

// the maximum time before unverified asset is expired
const (
	timeout = 60 * time.Minute
)

// to control expiry
type expiry struct {
	assetIndex transactionrecord.AssetIndex // item to remove
	expires    time.Time                    // remove the record after this time
}

// expiry loop
func (state *expiryData) Run(args interface{}, shutdown <-chan struct{}) {

	log := state.log

	l := list.New()
	delay := time.After(time.Minute)
loop:
	for {
		log.Info("waiting…")
		select {
		case <-shutdown:
			break loop
		case assetIndex := <-state.queue:
			log.Infof("received: asset index: %s", assetIndex)
			l.PushBack(expiry{
				assetIndex: assetIndex,
				expires:    time.Now().Add(timeout),
			})
		case <-delay:
			for {
				e := l.Front()
				if nil == e {
					delay = time.After(time.Minute)
					break
				}
				item := e.Value.(expiry)
				d := time.Since(item.expires)
				if d < 0 {
					delay = time.After(-d)
					break
				}
				l.Remove(e)

				globalData.Lock()
				cache, ok := globalData.cache[item.assetIndex]
				if ok {
					switch cache.state {
					case pendingState:
						cache.state = expiringState
						item.expires = time.Now().Add(timeout)
						l.PushBack(item)
					case expiringState:
						log.Infof("expired: asset index: %s", item.assetIndex)
						delete(globalData.cache, item.assetIndex)
					case verifiedState:
						// the item just dropped from expiry queue
						// but still exists in the map
					default:
						log.Criticalf("expired: invalid cache state: %d for: %s", cache.state, item.assetIndex)
					}
				}
				globalData.Unlock()
			}
		}
	}
}
