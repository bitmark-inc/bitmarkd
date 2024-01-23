// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package peer

import (
	"encoding/binary"

	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/asset"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/payment"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
)

// process received data
func processSubscription(log *logger.L, command string, arguments [][]byte) {

	dataLength := len(arguments)
	switch command {
	case "block":
		if dataLength < 1 {
			log.Debugf("block with too few data: %d items", dataLength)
			return
		}
		log.Infof("received block: %x", arguments[0])
		if !mode.Is(mode.Normal) {
			err := fault.NotAvailableDuringSynchronise
			log.Debugf("failed assets: error: %s", err)
		} else {
			messagebus.Bus.Blockstore.Send("remote", arguments[0])
		}

	case "assets":
		if dataLength < 1 {
			log.Debugf("assets with too few data: %d items", dataLength)
			return
		}
		log.Infof("received assets: %x", arguments[0])
		err := processAssets(arguments[0])
		if err != nil {
			log.Debugf("failed assets: error: %s", err)
		} else {
			messagebus.Bus.Broadcast.Send("assets", arguments[0])
		}

	case "issues":
		if dataLength < 1 {
			log.Debugf("issues with too few data: %d items", dataLength)
			return
		}
		log.Infof("received issues: %x", arguments[0])
		err := processIssues(arguments[0])
		if err != nil {
			log.Debugf("failed issues: error: %s", err)
		} else {
			messagebus.Bus.Broadcast.Send("issues", arguments[0])
		}

	case "transfer":
		if dataLength < 1 {
			log.Debugf("transfer with too few data: %d items", dataLength)
			return
		}
		log.Infof("received transfer: %x", arguments[0])
		err := processTransfer(arguments[0])
		if err != nil {
			log.Debugf("failed transfer: error: %s", err)
		} else {
			messagebus.Bus.Broadcast.Send("transfer", arguments[0])
		}

	case "proof":
		if dataLength < 1 {
			log.Debugf("proof with too few data: %d items", dataLength)
			return
		}
		log.Infof("received proof: %x", arguments[0])
		err := processProof(arguments[0])
		if err != nil {
			log.Debugf("failed proof: error: %s", err)
		} else {
			messagebus.Bus.Broadcast.Send("proof", arguments[0])
		}

	case "rpc":
		if dataLength < 3 {
			log.Debugf("rpc with too few data: %d items", dataLength)
			return
		}
		if len(arguments[2]) != 8 {
			log.Debug("rpc with invalid timestamp")
			return
		}
		timestamp := binary.BigEndian.Uint64(arguments[2])
		log.Infof("received rpc: fingerprint: %x  rpc: %x  timestamp: %d", arguments[0], arguments[1], timestamp)
		if announce.AddRPC(arguments[0], arguments[1], timestamp) {
			messagebus.Bus.Broadcast.Send("rpc", arguments[0:3]...)
		}

	case "peer":
		if dataLength < 3 {
			log.Debugf("peer with too few data: %d items", dataLength)
			return
		}
		if len(arguments[2]) != 8 {
			log.Debug("peer with invalid timestamp")
			return
		}
		timestamp := binary.BigEndian.Uint64(arguments[2])
		log.Infof("received peer: %x  listener: %x  timestamp: %d", arguments[0], arguments[1], timestamp)
		if announce.AddPeer(arguments[0], arguments[1], timestamp) {
			messagebus.Bus.Broadcast.Send("peer", arguments[0:3]...)
		}

	default:
		log.Debugf("received unhandled command: %q arguments: %x", command, arguments)

	}
}

// un pack each asset and cache them
func processAssets(packed []byte) error {

	if len(packed) == 0 {
		return fault.MissingParameters
	}

	if !mode.Is(mode.Normal) {
		return fault.NotAvailableDuringSynchronise
	}

	ok := false
	for len(packed) != 0 {
		transaction, n, err := transactionrecord.Packed(packed).Unpack(mode.IsTesting())
		if err != nil {
			return err
		}

		switch tx := transaction.(type) {
		case *transactionrecord.AssetData:
			_, packedAsset, err := asset.Cache(tx, storage.Pool.Assets)
			if err != nil {
				return err
			}
			if packedAsset != nil {
				ok = true
			}

		default:
			return fault.TransactionIsNotAnAsset
		}
		packed = packed[n:]
	}

	// all items were duplicates
	if !ok {
		return fault.NoNewTransactions
	}
	return nil
}

// un pack each issue and cache them
func processIssues(packed []byte) error {

	if len(packed) == 0 {
		return fault.MissingParameters
	}

	if !mode.Is(mode.Normal) {
		return fault.NotAvailableDuringSynchronise
	}

	packedIssues := transactionrecord.Packed(packed)
	issueCount := 0 // for payment difficulty

	issues := make([]*transactionrecord.BitmarkIssue, 0, reservoir.MaximumIssues)
	for len(packedIssues) != 0 {
		transaction, n, err := packedIssues.Unpack(mode.IsTesting())
		if err != nil {
			return err
		}

		switch tx := transaction.(type) {
		case *transactionrecord.BitmarkIssue:
			issues = append(issues, tx)
			issueCount += 1
		default:
			return fault.TransactionIsNotAnIssue
		}
		packedIssues = packedIssues[n:]
	}
	if len(issues) == 0 {
		return fault.MissingParameters
	}

	rsvr := reservoir.Get()
	_, duplicate, err := rsvr.StoreIssues(issues)
	if err != nil {
		return err
	}

	if duplicate {
		return fault.TransactionAlreadyExists
	}

	return nil
}

// unpack transfer and process it
func processTransfer(packed []byte) error {

	if len(packed) == 0 {
		return fault.MissingParameters
	}

	if !mode.Is(mode.Normal) {
		return fault.NotAvailableDuringSynchronise
	}

	transaction, _, err := transactionrecord.Packed(packed).Unpack(mode.IsTesting())
	if err != nil {
		return err
	}

	duplicate := false

	transfer, ok := transaction.(transactionrecord.BitmarkTransfer)

	rsvr := reservoir.Get()
	if ok {

		_, duplicate, err = rsvr.StoreTransfer(transfer)

	} else {
		switch tx := transaction.(type) {

		case *transactionrecord.ShareGrant:
			_, duplicate, err = rsvr.StoreGrant(tx)

		case *transactionrecord.ShareSwap:
			_, duplicate, err = rsvr.StoreSwap(tx)

		default:
			return fault.TransactionIsNotATransfer
		}
	}

	if err != nil {
		return err
	}

	if duplicate {
		return fault.TransactionAlreadyExists
	}

	return nil
}

// process proof block
func processProof(packed []byte) error {

	if len(packed) == 0 {
		return fault.MissingParameters
	}

	if !mode.Is(mode.Normal) {
		return fault.NotAvailableDuringSynchronise
	}

	var payId pay.PayId
	nonceLength := len(packed) - len(payId) // could be negative
	if nonceLength < payment.MinimumNonceLength || nonceLength > payment.MaximumNonceLength {
		return fault.InvalidNonce
	}

	copy(payId[:], packed[:len(payId)])
	nonce := packed[len(payId):]
	rsvr := reservoir.Get()
	status := rsvr.TryProof(payId, nonce)
	if reservoir.TrackingAccepted != status {
		// pay id already processed or was invalid
		return fault.PayIdAlreadyUsed
	}

	return nil
}
