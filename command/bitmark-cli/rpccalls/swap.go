// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpccalls

import (
	"encoding/hex"

	"golang.org/x/crypto/ed25519"

	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/keypair"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/rpc"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
)

var (
	ErrMakeSwapFail  = fault.ProcessError("make swap failed")
	ErrNotSwapRecord = fault.InvalidError("not swap record")
)

type SwapData struct {
	ShareIdOne  string
	QuantityOne uint64
	OwnerOne    *keypair.KeyPair
	ShareIdTwo  string
	QuantityTwo uint64
	OwnerTwo    *keypair.KeyPair
	BeforeBlock uint64
}

type SwapCountersignData struct {
	Swap      string
	Recipient *keypair.KeyPair
}

// JSON data to output after swap completes
type SwapReply struct {
	SwapId   merkle.Digest                                   `json:"swapId"`
	PayId    pay.PayId                                       `json:"payId"`
	Payments map[string]transactionrecord.PaymentAlternative `json:"payments"`
	Commands map[string]string                               `json:"commands,omitempty"`
}

type SwapSingleSignedReply struct {
	Identity string `json:"identity"`
	Swap     string `json:"swap"`
}

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
		swapConfig.BeforeBlock = info.Blocks.Height + 100 // allow plenty of time to mine
	}

	packed, swap, err := makeSwapOneSignature(client.testnet, shareIdOne, swapConfig.QuantityOne, swapConfig.OwnerOne, shareIdTwo, swapConfig.QuantityTwo, swapConfig.OwnerTwo, swapConfig.BeforeBlock)
	if nil != err {
		return nil, err
	}
	if nil == swap {
		return nil, ErrMakeSwapFail
	}

	client.printJson("Swap Request", swap)

	response := SwapSingleSignedReply{
		Identity: swap.OwnerOne.String(),
		Swap:     hex.EncodeToString(packed),
	}

	return &response, nil
}

func (client *Client) CountersignSwap(swap *transactionrecord.ShareSwap) (*SwapReply, error) {

	client.printJson("Swap Request", swap)

	var reply rpc.ShareSwapReply
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

func makeSwapOneSignature(testnet bool, shareIdOne merkle.Digest, quantityOne uint64, ownerOne *keypair.KeyPair, shareIdTwo merkle.Digest, quantityTwo uint64, ownerTwo *keypair.KeyPair, beforeBlock uint64) ([]byte, *transactionrecord.ShareSwap, error) {

	ownerOneAddress := makeAddress(ownerOne, testnet)
	ownerTwoAddress := makeAddress(ownerTwo, testnet)

	r := transactionrecord.ShareSwap{
		ShareIdOne:       shareIdOne,
		QuantityOne:      quantityOne,
		OwnerOne:         ownerOneAddress,
		ShareIdTwo:       shareIdTwo,
		QuantityTwo:      quantityTwo,
		OwnerTwo:         ownerTwoAddress,
		BeforeBlock:      beforeBlock,
		Signature:        nil,
		Countersignature: nil,
	}

	// pack without signature
	packed, err := r.Pack(ownerOneAddress)
	if nil == err {
		return nil, nil, ErrMakeSwapFail
	} else if fault.ErrInvalidSignature != err {
		return nil, nil, err
	}

	// attach signature
	signature := ed25519.Sign(ownerOne.PrivateKey, packed)
	r.Signature = signature[:]

	// include first signature by packing again
	packed, err = r.Pack(ownerOneAddress)
	if nil == err {
		return nil, nil, ErrMakeSwapFail
	} else if fault.ErrInvalidSignature != err {
		return nil, nil, err
	}
	return packed, &r, nil
}
