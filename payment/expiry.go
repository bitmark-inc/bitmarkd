// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package payment

import (
	"container/list"
	"time"
)

// the maximum time before either a payment track or proof is received
// if the timeout is reached then the transactions are dropped
const (
	timeout = 15 * time.Minute
)

// to control expiry
type expiry struct {
	payId   PayId     // item to remove
	expires time.Time // remove the record after this time
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
				expires: time.Now().Add(timeout),
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
