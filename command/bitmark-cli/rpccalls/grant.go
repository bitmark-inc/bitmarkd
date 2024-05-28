// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpccalls

import (
	"encoding/hex"

	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/configuration"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/rpc/share"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"golang.org/x/crypto/ed25519"
)

// GrantData - data for a grant request
type GrantData struct {
	ShareId     string
	Quantity    uint64
	Owner       *configuration.Private
	Recipient   *account.Account
	BeforeBlock uint64
}

// GrantCountersignData - data to be countersigned
type GrantCountersignData struct {
	Grant     string
	Recipient *configuration.Private
}

// GrantReply - JSON data to output after grant completes
type GrantReply struct {
	GrantId  merkle.Digest                                   `json:"grantId"`
	PayId    pay.PayId                                       `json:"payId"`
	Payments map[string]transactionrecord.PaymentAlternative `json:"payments"`
	Commands map[string]string                               `json:"commands,omitempty"`
}

// GrantSingleSignedReply - result from single signature
type GrantSingleSignedReply struct {
	Identity string `json:"identity"`
	Grant    string `json:"grant"`
}

// Grant - perform the grant request
func (client *Client) Grant(grantConfig *GrantData) (*GrantSingleSignedReply, error) {

	var shareId merkle.Digest
	err := shareId.UnmarshalText([]byte(grantConfig.ShareId))
	if err != nil {
		return nil, err
	}

	if grantConfig.BeforeBlock == 0 {
		info, err := client.GetBitmarkInfo()
		if err != nil {
			return nil, err
		}
		grantConfig.BeforeBlock = info.Block.Height + 100 // allow plenty of time to mine
	}

	packed, grant, err := makeGrantOneSignature(client.testnet, shareId, grantConfig.Quantity, grantConfig.Owner, grantConfig.Recipient, grantConfig.BeforeBlock)
	if err != nil {
		return nil, err
	}
	if grant == nil {
		return nil, fault.MakeGrantFailed
	}

	client.printJson("Grant Request", grant)

	response := GrantSingleSignedReply{
		Identity: grant.Owner.String(),
		Grant:    hex.EncodeToString(packed),
	}

	return &response, nil
}

// CountersignGrant - perform the countersignature
func (client *Client) CountersignGrant(grant *transactionrecord.ShareGrant) (*GrantReply, error) {

	client.printJson("Grant Request", grant)

	var reply share.GrantReply
	err := client.client.Call("Share.Grant", grant, &reply)
	if err != nil {
		return nil, err
	}

	tpid, err := reply.PayId.MarshalText()
	if err != nil {
		return nil, err
	}

	commands := make(map[string]string)
	for _, payment := range reply.Payments {
		currency := payment[0].Currency
		commands[currency.String()] = paymentCommand(client.testnet, currency, string(tpid), payment)
	}

	client.printJson("Grant Reply", reply)

	// make response
	response := GrantReply{
		GrantId:  reply.TxId,
		PayId:    reply.PayId,
		Payments: reply.Payments,
		Commands: commands,
	}

	return &response, nil
}

func makeGrantOneSignature(testnet bool, shareId merkle.Digest, quantity uint64, owner *configuration.Private, recipient *account.Account, beforeBlock uint64) ([]byte, *transactionrecord.ShareGrant, error) {

	ownerAccount := owner.PrivateKey.Account()

	r := transactionrecord.ShareGrant{
		ShareId:          shareId,
		Quantity:         quantity,
		Owner:            ownerAccount,
		Recipient:        recipient,
		BeforeBlock:      beforeBlock,
		Signature:        nil,
		Countersignature: nil,
	}

	// pack without signature
	packed, err := r.Pack(ownerAccount)
	if err == nil {
		return nil, nil, fault.MakeGrantFailed
	} else if fault.InvalidSignature != err {
		return nil, nil, err
	}

	// attach signature
	r.Signature = ed25519.Sign(owner.PrivateKey.PrivateKeyBytes(), packed)

	// include first signature by packing again
	packed, err = r.Pack(ownerAccount)
	if err == nil {
		return nil, nil, fault.MakeGrantFailed
	} else if fault.InvalidSignature != err {
		return nil, nil, err
	}
	return packed, &r, nil
}
