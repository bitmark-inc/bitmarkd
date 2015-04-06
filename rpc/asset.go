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

	packedAsset, err := arguments.Pack(arguments.Registrant)
	if nil != err {
		return err
	}

	log.Debugf("Asset packed tx: %x", packedAsset)

	reply.AssetIndex = arguments.AssetIndex()
	_, _, found := reply.AssetIndex.Read()

	id, exists := packedAsset.Exists()

	reply.Duplicate = found || exists
	reply.TxId = id            // this could be the id of an earlier version of the same asset
	reply.PaymentAddress = nil // no payment for asset

	// announce transaction to system
	if !found {
		messagebus.Send(packedAsset)
	}

	log.Infof("Asset.Register found: %v", found)
	log.Infof("Asset.Register exists: %v", exists)
	return nil
}
