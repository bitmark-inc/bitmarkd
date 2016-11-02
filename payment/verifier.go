// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package payment

import (
	"github.com/bitmark-inc/bitmarkd/reservoir"
)

// verifier loop
func (state *verifierData) Run(args interface{}, shutdown <-chan struct{}) {

	log := state.log
	globalData.log.Info("starting…")

loop:
	for {
		log.Info("waiting…")
		select {
		case <-shutdown:
			break loop
		case payId := <-state.queue:
			log.Infof("received: pay id: %s", payId)
			reservoir.SetVerified(payId)
		}
	}
}
