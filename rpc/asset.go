// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	//"github.com/bitmark-inc/bitmarkd/block"
	//"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
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
	AssetIndex transactionrecord.AssetIndex `json:"asset"`
	Duplicate  bool                         `json:"duplicate"`
	Err        string                       `json:"error,omitempty"`
}

func (asset *Asset) Register(arguments *transactionrecord.AssetData, reply *AssetRegisterReply) error {
	log := asset.log

	log.Infof("Asset.Register: %s", arguments.Fingerprint)
	log.Infof("Asset.Register: %v", arguments)

	reply.AssetIndex = arguments.AssetIndex()
	// _, id, found := reply.AssetIndex.Read()

	found := false // ***** FIX THIS: temporary for testing

	if !found {
		packedAsset, err := arguments.Pack(arguments.Registrant)
		if nil != err {
			return err
		}
		//messagebus.Send("", packedAsset) // ***** FIX THIS: need to resore broadcast

		log.Debugf("Sent asset packed tx: %x", packedAsset)

		// get this tx id value - could be changed by later asset being mined
		//***** FIX THIS: is this needed
		//id = packedAsset.MakeLink()
	}

	reply.Duplicate = found

	log.Infof("Asset.Register found: %v", found)
	return nil
}
