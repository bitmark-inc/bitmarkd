// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package reservoir

import (
	"bytes"
	"github.com/bitmark-inc/bitmarkd/asset"
	"github.com/bitmark-inc/bitmarkd/cache"
	"github.com/bitmark-inc/bitmarkd/constants"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
	"time"
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

func fetchAsset(assetId transactionrecord.AssetIdentifier) ([]byte, error) {
	packedAsset := asset.Get(assetId)
	if nil == packedAsset {
		return nil, fault.ErrAssetNotFound
	}

	unpacked, _, err := packedAsset.Unpack(mode.IsTesting())
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

	log.Info("Start rebroadcasting local transactions…")

unverified_tx:
	for _, val := range cache.Pool.UnverifiedTxEntries.Items() {
		item := val.(*unverifiedItem)
		if item.links != nil {
			messagebus.Bus.Broadcast.Send("transfer", item.transactions[0])
		} else {
			packedAssets := []byte{}
			for assetId, _ := range item.itemData.assetIds {
				packedAsset, err := fetchAsset(assetId)
				if fault.ErrAssetNotFound == err {
					// asset was confirmed in an earlier block
				} else if err != nil {
					log.Errorf("asset id: %s  error: %s", assetId, err)
					continue unverified_tx // skip the corresponding issue since asset is corrupt
				} else {
					packedAssets = append(packedAssets, packedAsset...)
				}
			}
			if len(packedAssets) > 0 {
				messagebus.Bus.Broadcast.Send("assets", packedAssets)
			}
			messagebus.Bus.Broadcast.Send("issues", bytes.Join(item.itemData.transactions, []byte{}))
		}
	}

verified_tx:
	for _, val := range cache.Pool.VerifiedTx.Items() {
		v := val.(*verifiedItem)

		if nil != v.itemData.links { // single transfer

			messagebus.Bus.Broadcast.Send("transfer", v.transaction)

		} else if 0 == v.index {
			// first of verified block so recreate whole issue block
			// to get same pay id

			packedAssets := []byte{}
			for assetId, _ := range v.itemData.assetIds {
				packedAsset, err := fetchAsset(assetId)
				if fault.ErrAssetNotFound == err {
					// asset was confirmed in an earlier block
				} else if err != nil {
					log.Errorf("asset id: %s  error: %s", assetId, err)
					continue verified_tx // skip the corresponding issue since asset is corrupt
				} else {
					packedAssets = append(packedAssets, packedAsset...)
				}
			}
			if len(packedAssets) > 0 {
				messagebus.Bus.Broadcast.Send("assets", packedAssets)
			}

			messagebus.Bus.Broadcast.Send("issues", bytes.Join(v.itemData.transactions, []byte{}))

			// recreate the proof message
			payId := pay.NewPayId(v.itemData.transactions)
			messagebus.Bus.Broadcast.Send("proof", append(payId[:], v.itemData.nonce...))
		}
	}

	globalData.RUnlock()
}
