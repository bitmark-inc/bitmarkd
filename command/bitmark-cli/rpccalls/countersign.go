// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpccalls

import (
	"encoding/hex"

	"golang.org/x/crypto/ed25519"

	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/configuration"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
)

// CountersignData - data for a countersignature request
type CountersignData struct {
	Transaction string
	NewOwner    *configuration.Private
}

// Countersign - countersign a transfer
func (client *Client) Countersign(countersignConfig *CountersignData) (interface{}, error) {

	b, err := hex.DecodeString(countersignConfig.Transaction)
	if nil != err {
		return nil, err
	}

	bCs := append(b, 0x01, 0x00) // one-byte countersignature to allow unpack to succeed
	r, _, err := transactionrecord.Packed(bCs).Unpack(client.testnet)
	if nil != err {
		return nil, err
	}

	// attach signature
	signature := ed25519.Sign(countersignConfig.NewOwner.PrivateKey.PrivateKeyBytes(), b)

	switch tx := r.(type) {
	case *transactionrecord.BitmarkTransferCountersigned:
		tx.Countersignature = signature[:]
		return client.CountersignTransfer(tx)

	case *transactionrecord.BlockOwnerTransfer:
		tx.Countersignature = signature[:]
		return client.CountersignBlockTransfer(tx)

	case *transactionrecord.ShareGrant:
		tx.Countersignature = signature[:]
		return client.CountersignGrant(tx)

	case *transactionrecord.ShareSwap:
		tx.Countersignature = signature[:]
		return client.CountersignSwap(tx)

	default:
		return nil, fault.NotACountersignableRecord
	}
}
