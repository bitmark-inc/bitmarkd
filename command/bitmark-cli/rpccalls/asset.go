// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpccalls

import (
	"fmt"

	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/configuration"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/rpc/assets"
	"github.com/bitmark-inc/bitmarkd/rpc/bitmarks"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"golang.org/x/crypto/ed25519"
)

// AssetData - asset data for bitmark creation
type AssetData struct {
	Name        string
	Metadata    string
	Quantity    int
	Registrant  *configuration.Private
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

	getArgs := assets.GetArguments{
		Fingerprints: []string{assetConfig.Fingerprint},
	}

	client.printJson("Asset Get Request", getArgs)

	var getReply assets.GetReply
	if err := client.client.Call("Assets.Get", &getArgs, &getReply); err != nil {
		return nil, err
	}

	if len(getReply.Assets) != 1 {
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
		if err != nil {
			return nil, err
		}
		result.AssetId = ai
		result.Confirmed = getReply.Assets[0].Confirmed

	default:
		if getReply.Assets[0].Data != nil {
			return nil, fmt.Errorf("non-asset response")
		}
	}

	client.printJson("Asset Get Reply", getReply)

	registrant := assetConfig.Registrant.PrivateKey.Account()
	r := transactionrecord.AssetData{
		Name:        assetConfig.Name,
		Fingerprint: assetConfig.Fingerprint,
		Metadata:    assetConfig.Metadata,
		Registrant:  registrant,
		Signature:   nil,
	}

	// pack without signature
	packed, err := r.Pack(registrant)
	if fault.InvalidSignature != err {
		return nil, err
	}

	// manually sign the record and attach signature
	r.Signature = ed25519.Sign(assetConfig.Registrant.PrivateKey.PrivateKeyBytes(), packed)

	// check that signature is correct by packing again
	if _, err = r.Pack(registrant); err != nil {
		return nil, err
	}

	client.printJson("Asset Request", r)

	args := bitmarks.CreateArguments{
		Assets: []*transactionrecord.AssetData{&r},
		Issues: nil,
	}

	var reply bitmarks.CreateReply
	if err := client.client.Call("Bitmarks.Create", &args, &reply); err != nil {
		return nil, err
	}

	client.printJson("Asset Reply", reply)

	result.AssetId = reply.Assets[0].AssetId
	return result, nil
}
