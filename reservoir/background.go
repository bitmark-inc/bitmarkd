// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package reservoir

import (
	"time"

	"github.com/bitmark-inc/bitmarkd/asset"
	"github.com/bitmark-inc/bitmarkd/cache"
	"github.com/bitmark-inc/bitmarkd/constants"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/logger"
)

type rebroadcaster struct {
	log *logger.L
}

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
	for _, val := range cache.Pool.UnverifiedTxEntries.Items() {
		item := val.(*unverifiedItem)
		if item.links != nil {
			packedTransfer = append(packedTransfer, item.transactions[0])
		}
	}

	hadAsset := make(map[transactionrecord.AssetIndex]struct{})
	for _, val := range cache.Pool.VerifiedTx.Items() {
		v := val.(*verifiedItem)
		if v.itemData.links == nil {
			packedIssue := transactionrecord.Packed(v.transaction)
			assetId := v.itemData.assetIds[v.index]
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
