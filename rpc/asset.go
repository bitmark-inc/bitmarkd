// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/transaction"
	"github.com/bitmark-inc/logger"
)

// Asset type
// ----------

type Asset struct {
	log *logger.L
}

// Asset registration
// ------------------

type AssetRegisterReply struct {
	TxId           transaction.Link       `json:"txid"`
	AssetIndex     transaction.AssetIndex `json:"asset"`
	PaymentAddress []block.MinerAddress   `json:"paymentAddress"`
	Duplicate      bool                   `json:"duplicate"`
	Err            string                 `json:"error,omitempty"`
}

func (asset *Asset) Register(arguments *transaction.AssetData, reply *AssetRegisterReply) error {
	log := asset.log

	log.Infof("Asset.Register: %s", arguments.Fingerprint)
	log.Infof("Asset.Register: %v", arguments)

	reply.AssetIndex = arguments.AssetIndex()
	_, id, found := reply.AssetIndex.Read()

	if !found {
		packedAsset, err := arguments.Pack(arguments.Registrant)
		if nil != err {
			return err
		}
		messagebus.Send("", packedAsset)

		log.Debugf("Sent asset packed tx: %x", packedAsset)

		// get this tx id value - could be changed by later asset being mined
		id = packedAsset.MakeLink()
	}

	reply.Duplicate = found
	reply.TxId = id            // this could be the id of an earlier version of the same asset
	reply.PaymentAddress = nil // no payment for asset

	log.Infof("Asset.Register found: %v", found)
	return nil
}
