// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package asset

import (
	"container/list"
	"github.com/bitmark-inc/bitmarkd/constants"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"time"
)

// to control expiry
type expiry struct {
	assetId transactionrecord.AssetIdentifier // item to remove
	expires time.Time                         // remove the record after this time
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
		case assetId := <-state.queue:
			log.Infof("received: asset id: %s", assetId)
			l.PushBack(expiry{
				assetId: assetId,
				expires: time.Now().Add(constants.AssetTimeout),
			})
		case <-delay:
		inner_loop:
			for {
				e := l.Front()
				if nil == e {
					delay = time.After(time.Minute)
					break inner_loop
				}
				item := e.Value.(expiry)
				d := time.Since(item.expires)
				if d < 0 {
					delay = time.After(-d)
					break inner_loop
				}
				l.Remove(e)

				globalData.Lock()
				cache, ok := globalData.cache[item.assetId]
				if ok {
					switch cache.state {
					case pendingState:
						cache.state = expiringState
						item.expires = time.Now().Add(constants.AssetTimeout)
						l.PushBack(item)
					case expiringState:
						log.Infof("expired: asset id: %s", item.assetId)
						delete(globalData.cache, item.assetId)
					case verifiedState:
						// the item just dropped from expiry queue
						// but still exists in the map
					default:
						log.Criticalf("expired: invalid cache state: %d for: %s", cache.state, item.assetId)
					}
				}
				globalData.Unlock()
			}
		}
	}
	log.Info("finished")
}
