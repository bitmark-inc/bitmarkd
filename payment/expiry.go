// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package payment

import (
	"container/list"
	"github.com/bitmark-inc/bitmarkd/constants"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"time"
)

// to control expiry
type expiry struct {
	payId   reservoir.PayId // item to remove
	expires time.Time       // remove the record after this time
}

// expiry loop
func (state *expiryData) Run(args interface{}, shutdown <-chan struct{}) {

	log := state.log
	globalData.log.Info("starting…")

	l := list.New()
	delay := time.After(time.Minute)
loop:
	for {
		log.Info("waiting…")
		select {
		case <-shutdown:
			break loop
		case payId := <-state.queue:
			log.Infof("received: pay id: %s", payId)
			l.PushBack(expiry{
				payId:   payId,
				expires: time.Now().Add(constants.PaymentTimeout),
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
				log.Infof("expired: pay id: %s", item.payId)
				remove(item.payId)
				l.Remove(e)
			}
		}
	}
}
