// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package payment

import (
// "github.com/bitmark-inc/bitmarkd/datastore"
// "github.com/bitmark-inc/bitmarkd/fault"
//"time"
)

// verifier loop
func (state *verifierData) Run(args interface{}, shutdown <-chan struct{}) {

	log := state.log

loop:
	for {
		log.Info("waiting…")
		select {
		case <-shutdown:
			break loop
		case transactions := <-state.queue:
			log.Infof("received: transactions: %x", transactions)
			state.setVerified(transactions)
		}
	}
}

// store all transactions in disk storage to await confirmation
func (state *verifierData) setVerified(transactions []byte) {
	// ***** FIX THIS: add the verifiaction process
	// ***** FIX THIS: add code here
	state.log.Errorf("***** FIX THIS: received: transactions: %x", transactions)
}
