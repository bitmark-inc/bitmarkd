// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpccalls

import (
	"encoding/hex"

	"golang.org/x/crypto/ed25519"

	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/configuration"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/rpc/share"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
)

// SwapData - data for a swap request
type SwapData struct {
	ShareIdOne  string
	QuantityOne uint64
	OwnerOne    *configuration.Private
	ShareIdTwo  string
	QuantityTwo uint64
	OwnerTwo    *account.Account
	BeforeBlock uint64
}

// SwapCountersignData - data for countersigning
type SwapCountersignData struct {
	Swap      string
	Recipient *configuration.Private
}

// SwapReply - JSON data to output after swap completes
type SwapReply struct {
	SwapId   merkle.Digest                                   `json:"swapId"`
	PayId    pay.PayId                                       `json:"payId"`
	Payments map[string]transactionrecord.PaymentAlternative `json:"payments"`
	Commands map[string]string                               `json:"commands,omitempty"`
}

// SwapSingleSignedReply - result of single signature
type SwapSingleSignedReply struct {
	Identity string `json:"identity"`
	Swap     string `json:"swap"`
}

// Swap - perform swap request
func (client *Client) Swap(swapConfig *SwapData) (*SwapSingleSignedReply, error) {

	var shareIdOne merkle.Digest
	err := shareIdOne.UnmarshalText([]byte(swapConfig.ShareIdOne))
	if nil != err {
		return nil, err
	}

	var shareIdTwo merkle.Digest
	err = shareIdTwo.UnmarshalText([]byte(swapConfig.ShareIdTwo))
	if nil != err {
		return nil, err
	}

	if 0 == swapConfig.BeforeBlock {
		info, err := client.GetBitmarkInfo()
		if nil != err {
			return nil, err
		}
		swapConfig.BeforeBlock = info.Block.Height + 100 // allow plenty of time to mine
	}

	packed, swap, err := makeSwapOneSignature(client.testnet, shareIdOne, swapConfig.QuantityOne, swapConfig.OwnerOne, shareIdTwo, swapConfig.QuantityTwo, swapConfig.OwnerTwo, swapConfig.BeforeBlock)
	if nil != err {
		return nil, err
	}
	if nil == swap {
		return nil, fault.MakeSwapFailed
	}

	client.printJson("Swap Request", swap)

	response := SwapSingleSignedReply{
		Identity: swap.OwnerOne.String(),
		Swap:     hex.EncodeToString(packed),
	}

	return &response, nil
}

// CountersignSwap - perform countersigning
func (client *Client) CountersignSwap(swap *transactionrecord.ShareSwap) (*SwapReply, error) {

	client.printJson("Swap Request", swap)

	var reply share.SwapReply
	err := client.client.Call("Share.Swap", swap, &reply)
	if nil != err {
		return nil, err
	}

	tpid, err := reply.PayId.MarshalText()
	if nil != err {
		return nil, err
	}

	commands := make(map[string]string)
	for _, payment := range reply.Payments {
		currency := payment[0].Currency
		commands[currency.String()] = paymentCommand(client.testnet, currency, string(tpid), payment)
	}

	client.printJson("Swap Reply", reply)

	// make response
	response := SwapReply{
		SwapId:   reply.TxId,
		PayId:    reply.PayId,
		Payments: reply.Payments,
		Commands: commands,
	}

	return &response, nil
}

func makeSwapOneSignature(testnet bool, shareIdOne merkle.Digest, quantityOne uint64, ownerOne *configuration.Private, shareIdTwo merkle.Digest, quantityTwo uint64, ownerTwo *account.Account, beforeBlock uint64) ([]byte, *transactionrecord.ShareSwap, error) {

	ownerOneAccount := ownerOne.PrivateKey.Account()

	r := transactionrecord.ShareSwap{
		ShareIdOne:       shareIdOne,
		QuantityOne:      quantityOne,
		OwnerOne:         ownerOneAccount,
		ShareIdTwo:       shareIdTwo,
		QuantityTwo:      quantityTwo,
		OwnerTwo:         ownerTwo,
		BeforeBlock:      beforeBlock,
		Signature:        nil,
		Countersignature: nil,
	}

	// pack without signature
	packed, err := r.Pack(ownerOneAccount)
	if nil == err {
		return nil, nil, fault.MakeSwapFailed
	} else if fault.InvalidSignature != err {
		return nil, nil, err
	}

	// attach signature
	signature := ed25519.Sign(ownerOne.PrivateKey.PrivateKeyBytes(), packed)
	r.Signature = signature[:]

	// include first signature by packing again
	packed, err = r.Pack(ownerOneAccount)
	if nil == err {
		return nil, nil, fault.MakeSwapFailed
	} else if fault.InvalidSignature != err {
		return nil, nil, err
	}
	return packed, &r, nil
}
