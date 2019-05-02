// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package reservoir

import (
	"time"

	"github.com/bitmark-inc/bitmarkd/asset"
	"github.com/bitmark-inc/bitmarkd/constants"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
)

type rebroadcaster struct {
	log *logger.L
}

func (r *rebroadcaster) Run(args interface{}, shutdown <-chan struct{}) {

	r.log = logger.New("rebroadcaster")
	log := r.log

	log.Info("starting…")

loop:
	for {
		log.Debug("waiting…")
		select {
		case <-shutdown:
			log.Info("shutting down…")
			break loop
		case <-time.After(constants.RebroadcastInterval): // timeout
			r.process()
		}
	}

	log.Info("stopped")
}

// process all pending and verified transactions
func (r *rebroadcaster) process() {
	log := r.log
	globalData.RLock()

	log.Info("Start rebroadcasting local transactions…")

	// pending

	for _, item := range globalData.pendingTransactions {
		broadcastTransaction(item.tx)
	}
	for _, item := range globalData.pendingFreeIssues {
		broadcastFreeIssue(item)
	}
	for _, item := range globalData.pendingPaidIssues {
		broadcastPaidIssue(item)
	}

	// verified

	for _, item := range globalData.verifiedTransactions {
		broadcastTransaction(item)
	}
	for _, item := range globalData.verifiedFreeIssues {
		broadcastFreeIssue(item)
	}
	for _, item := range globalData.verifiedPaidIssues {
		broadcastPaidIssue(item)
	}

	globalData.RUnlock()
}

// send the transaction
func broadcastTransaction(item *transactionData) {
	messagebus.Bus.Broadcast.Send("transfer", item.packed)
}

// concatenate all transactions and send
func broadcastPaidIssue(item *issuePaymentData) {
	packedIssues := []byte{}
	for _, tx := range item.txs {
		packedIssues = append(packedIssues, tx.packed...)
	}
	messagebus.Bus.Broadcast.Send("issues", packedIssues)
}

// concatenate pending assets and issues, then send
// note there should not be any duplicate assets, i.e.
// 1. all issues are for the same asset
// 2. all issues are for different assets
func broadcastFreeIssue(item *issueFreeData) {

	packedAssets := []byte{}
	packedIssues := []byte{}

	for _, tx := range item.txs {
		assetId := tx.transaction.(*transactionrecord.BitmarkIssue).AssetId
		packedAsset := asset.Get(assetId)
		if nil != packedAsset {
			packedAssets = append(packedAssets, packedAsset...)
		}
		packedIssues = append(packedIssues, tx.packed...)
	}
	if len(packedAssets) > 0 {
		messagebus.Bus.Broadcast.Send("assets", packedAssets)
	}
	messagebus.Bus.Broadcast.Send("issues", packedIssues)

	// if the issue is a free issue, broadcast the proof
	if nil != item.difficulty {
		packed := make([]byte, len(item.payId), len(item.payId)+len(item.nonce))
		copy(packed, item.payId[:])
		packed = append(packed, item.nonce[:]...)
		messagebus.Bus.Broadcast.Send("proof", packed)
	}
}
