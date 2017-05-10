// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package reservoir

import (
	"github.com/bitmark-inc/bitmarkd/asset"
	"github.com/bitmark-inc/bitmarkd/constants"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/bitmarkd/util"
	"time"
)

// cycle time
const (
	verifierCycleTime = 5 * time.Minute
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

		case <-time.After(verifierCycleTime):
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

// rebroadcasting process
func (r *rebroadcaster) Run(args interface{}, shutdown <-chan struct{}) {

	log := r.log
	globalData := args.(*globalDataType)

	log.Info("starting…")

loop:
	for {
		log.Info("waiting…")
		select {
		case <-shutdown:
			log.Info("shutting down…")
			break loop
		case <-time.After(constants.RebroadcastInterval):
			r.process(globalData)
		}
	}
	log.Info("stopped")
}

func fetchAsset(assetId transactionrecord.AssetIndex) ([]byte, error) {
	packedAsset := asset.Get(assetId)
	if nil == packedAsset {
		return nil, fault.ErrAssetNotFound
	}

	unpacked, _, err := packedAsset.Unpack()
	if err != nil {
		return nil, err
	}

	_, ok := unpacked.(*transactionrecord.AssetData)
	if ok {
		return packedAsset[:], nil
	}
	return nil, fault.ErrTransactionIsNotAnAsset
}

func (r *rebroadcaster) process(globaldata *globalDataType) {
	log := r.log
	globalData.RLock()

	packedAssets := []byte{}
	packedIssues := [][]byte{}
	packedTransfer := [][]byte{}

	log.Info("Start rebroadcasting local transactions…")
	for _, item := range globalData.unverified.entries {
		if item.links != nil {
			packedTransfer = append(packedTransfer, item.transactions[0])
		}
	}

	hadAsset := make(map[transactionrecord.AssetIndex]struct{})
	for _, v := range globalData.verified {
		if v.data.links == nil {
			packedIssue := transactionrecord.Packed(v.transaction)
			assetId := v.data.assetIds[v.index]
			if _, ok := hadAsset[assetId]; !ok {
				packedAsset, err := fetchAsset(assetId)
				if fault.ErrAssetNotFound == err {
					// asset was confirmed in an earlier block
				} else if err != nil {
					log.Errorf("asset id[%d]: %s  error: %s", v.index, assetId, err)
					continue // skip the corresponding issue since asset is corrupt
				} else {
					packedAssets = append(packedAssets, packedAsset...)
				}
				hadAsset[assetId] = struct{}{}
			}
			packedIssues = append(packedIssues, packedIssue)
		} else {
			packedTransfer = append(packedTransfer, v.transaction)
		}
	}

	if len(packedAssets) != 0 {
		messagebus.Bus.Broadcast.Send("assets", packedAssets)
	}
	for _, issue := range packedIssues {
		messagebus.Bus.Broadcast.Send("issues", issue, util.ToVarint64(1))
	}
	for _, transfer := range packedTransfer {
		messagebus.Bus.Broadcast.Send("transfer", transfer)
	}
	globalData.RUnlock()
}
