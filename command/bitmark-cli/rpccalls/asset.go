// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpccalls

import (
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/keypair"
	"github.com/bitmark-inc/bitmarkd/rpc"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"golang.org/x/crypto/ed25519"
)

var (
	ErrAssetRequestFail = fault.ProcessError("send asset request failed")
)

type AssetData struct {
	Name        string
	Metadata    string
	Quantity    int
	Registrant  *keypair.KeyPair
	Fingerprint string
}

// build a properly signed asset
func (client *Client) MakeAsset(assetConfig *AssetData) (*transactionrecord.AssetIndex, error) {

	assetIndex := (*transactionrecord.AssetIndex)(nil)

	getArgs := rpc.AssetGetArguments{
		Fingerprints: []string{assetConfig.Fingerprint},
	}

	client.printJson("Asset Get Request", getArgs)

	var getReply rpc.AssetGetReply
	if err := client.client.Call("Assets.Get", &getArgs, &getReply); nil != err {
		return nil, err
	}

	if 1 != len(getReply.Assets) {
		return nil, ErrAssetRequestFail
	}

	switch getReply.Assets[0].Record {
	case "AssetData":
		ar, ok := getReply.Assets[0].Data.(map[string]interface{})
		if !ok {
			return nil, ErrAssetRequestFail
		}

		if ar["metadata"] != assetConfig.Metadata {
			return nil, ErrAssetRequestFail
		}
		if ar["name"] != assetConfig.Name {
			return nil, ErrAssetRequestFail
		}

		buffer, ok := getReply.Assets[0].AssetIndex.(string)
		if !ok {
			return nil, ErrAssetRequestFail
		}
		var ai transactionrecord.AssetIndex
		err := ai.UnmarshalText([]byte(buffer))
		if nil != err {
			return nil, err
		}
		assetIndex = &ai

	default:
		if nil != getReply.Assets[0].Data {
			return nil, ErrAssetRequestFail
		}
	}

	client.printJson("Asset Get Reply", getReply)

	if nil != assetIndex {
		return assetIndex, nil
	}

	registrantAddress := makeAddress(assetConfig.Registrant, client.testnet)

	r := transactionrecord.AssetData{
		Name:        assetConfig.Name,
		Fingerprint: assetConfig.Fingerprint,
		Metadata:    assetConfig.Metadata,
		Registrant:  registrantAddress,
		Signature:   nil,
	}

	// pack without signature
	packed, err := r.Pack(registrantAddress)
	if fault.ErrInvalidSignature != err {
		return nil, err
	}

	// manually sign the record and attach signature
	signature := ed25519.Sign(assetConfig.Registrant.PrivateKey, packed)
	r.Signature = signature[:]

	// check that signature is correct by packing again
	if _, err = r.Pack(registrantAddress); nil != err {
		return nil, err
	}

	client.printJson("Asset Request", r)

	args := rpc.CreateArguments{
		Assets: []*transactionrecord.AssetData{&r},
		Issues: nil,
	}

	var reply rpc.CreateReply
	if err := client.client.Call("Bitmarks.Create", &args, &reply); nil != err {
		return nil, err
	}

	client.printJson("Asset Reply", reply)

	return reply.Assets[0].AssetIndex, nil
}
