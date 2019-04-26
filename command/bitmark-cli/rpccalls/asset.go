// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpccalls

import (
	"fmt"

	"golang.org/x/crypto/ed25519"

	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/keypair"
	"github.com/bitmark-inc/bitmarkd/rpc"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
)

// AssetData - asset data for bitmark creation
type AssetData struct {
	Name        string
	Metadata    string
	Quantity    int
	Registrant  *keypair.KeyPair
	Fingerprint string
}

// AssetResult - result of an asset get request
type AssetResult struct {
	AssetId   *transactionrecord.AssetIdentifier
	Confirmed bool
}

// MakeAsset - build a properly signed asset
func (client *Client) MakeAsset(assetConfig *AssetData) (*AssetResult, error) {

	result := &AssetResult{}

	getArgs := rpc.AssetGetArguments{
		Fingerprints: []string{assetConfig.Fingerprint},
	}

	client.printJson("Asset Get Request", getArgs)

	var getReply rpc.AssetGetReply
	if err := client.client.Call("Assets.Get", &getArgs, &getReply); nil != err {
		return nil, err
	}

	if 1 != len(getReply.Assets) {
		return nil, fmt.Errorf("multple asset response")
	}

	switch getReply.Assets[0].Record {
	case "AssetData":
		ar, ok := getReply.Assets[0].Data.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("missing asset data")
		}

		if ar["metadata"] != assetConfig.Metadata {
			return nil, fmt.Errorf("mismatched asset metadata")
		}
		if ar["name"] != assetConfig.Name {
			return nil, fmt.Errorf("mismatched asset name")
		}

		buffer, ok := getReply.Assets[0].AssetId.(string)
		if !ok {
			return nil, fmt.Errorf("missing asset id")
		}

		ai := &transactionrecord.AssetIdentifier{}
		err := ai.UnmarshalText([]byte(buffer))
		if nil != err {
			return nil, err
		}
		result.AssetId = ai
		result.Confirmed = getReply.Assets[0].Confirmed

	default:
		if nil != getReply.Assets[0].Data {
			return nil, fmt.Errorf("non-asset response")
		}
	}

	client.printJson("Asset Get Reply", getReply)

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

	result.AssetId = reply.Assets[0].AssetId
	return result, nil
}
