// Copyright (c) 2014-2018 Bitmark Inc.
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
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
)

// process received data
func processSubscription(log *logger.L, command string, arguments [][]byte) {

	dataLength := len(arguments)
	switch string(command) {
	case "block":
		if dataLength < 1 {
			log.Warnf("block with too few data: %d items", dataLength)
			return
		}
		log.Infof("received block: %x", arguments[0])
		if !mode.Is(mode.Normal) {
			err := fault.ErrNotAvailableDuringSynchronise
			log.Warnf("failed assets: error: %s", err)
		} else {
			messagebus.Bus.Blockstore.Send("remote", arguments[0])
		}

	case "assets":
		if dataLength < 1 {
			log.Warnf("assets with too few data: %d items", dataLength)
			return
		}
		log.Infof("received assets: %x", arguments[0])
		err := processAssets(arguments[0])
		if nil != err {
			log.Warnf("failed assets: error: %s", err)
		} else {
			messagebus.Bus.Broadcast.Send("assets", arguments[0])
		}

	case "issues":
		if dataLength < 1 {
			log.Warnf("issues with too few data: %d items", dataLength)
			return
		}
		log.Infof("received issues: %x", arguments[0])
		err := processIssues(arguments[0])
		if nil != err {
			log.Warnf("failed issues: error: %s", err)
		} else {
			messagebus.Bus.Broadcast.Send("issues", arguments[0])
		}

	case "transfer":
		if dataLength < 1 {
			log.Warnf("transfer with too few data: %d items", dataLength)
			return
		}
		log.Infof("received transfer: %x", arguments[0])
		err := processTransfer(arguments[0])
		if nil != err {
			log.Warnf("failed transfer: error: %s", err)
		} else {
			messagebus.Bus.Broadcast.Send("transfer", arguments[0])
		}

	case "proof":
		if dataLength < 1 {
			log.Warnf("proof with too few data: %d items", dataLength)
			return
		}
		log.Infof("received proof: %x", arguments[0])
		err := processProof(arguments[0])
		if nil != err {
			log.Warnf("failed proof: error: %s", err)
		} else {
			messagebus.Bus.Broadcast.Send("proof", arguments[0])
		}

	case "rpc":
		if dataLength < 3 {
			log.Warnf("rpc with too few data: %d items", dataLength)
			return
		}
		timestamp := binary.BigEndian.Uint64(arguments[2])
		log.Infof("received rpc: fingerprint: %x  rpc: %x  timestamp: %d", arguments[0], arguments[1], timestamp)
		if announce.AddRPC(arguments[0], arguments[1], timestamp) {
			messagebus.Bus.Broadcast.Send("rpc", arguments[0:3]...)
		}

	case "peer":
		if dataLength < 3 {
			log.Warnf("peer with too few data: %d items", dataLength)
			return
		}
		timestamp := binary.BigEndian.Uint64(arguments[2])
		log.Infof("received peer: %x  listener: %x  timestamp: %d", arguments[0], arguments[1], timestamp)
		if announce.AddPeer(arguments[0], arguments[1], timestamp) {
			messagebus.Bus.Broadcast.Send("peer", arguments[0:3]...)
		}

	case "heart":
		if dataLength < 1 {
			log.Warnf("heart with too few data: %d items", dataLength)
			return
		}
		log.Infof("received heart: %q", arguments[0])
		// nothing to forward, this is just to keep communication alive

	default:
		log.Warnf("received unhandled command: %q arguments: %x", command, arguments)

	}
}

// un pack each asset and cache them
func processAssets(packed []byte) error {

	if 0 == len(packed) {
		return fault.ErrMissingParameters
	}

	if !mode.Is(mode.Normal) {
		return fault.ErrNotAvailableDuringSynchronise
	}

	ok := false
	for 0 != len(packed) {
		transaction, n, err := transactionrecord.Packed(packed).Unpack(mode.IsTesting())
		if nil != err {
			return err
		}

		switch tx := transaction.(type) {
		case *transactionrecord.AssetData:
			_, packedAsset, err := asset.Cache(tx)
			if nil != err {
				return err
			}
			if nil != packedAsset {
				ok = true
			}

		default:
			return fault.ErrTransactionIsNotAnAsset
		}
		packed = packed[n:]
	}

	if !ok {
		return fault.ErrNoNewTransactions
	}
	return nil
}

// un pack each issue and cache them
func processIssues(packed []byte) error {

	if 0 == len(packed) {
		return fault.ErrMissingParameters
	}

	if !mode.Is(mode.Normal) {
		return fault.ErrNotAvailableDuringSynchronise
	}

	packedIssues := transactionrecord.Packed(packed)
	issueCount := 0 // for payment difficulty

	issues := make([]*transactionrecord.BitmarkIssue, 0, 100)
	for 0 != len(packedIssues) {
		transaction, n, err := packedIssues.Unpack(mode.IsTesting())
		if nil != err {
			return err
		}

		switch tx := transaction.(type) {
		case *transactionrecord.BitmarkIssue:
			issues = append(issues, tx)
			issueCount += 1
		default:
			return fault.ErrTransactionIsNotAnIssue
		}
		packedIssues = packedIssues[n:]
	}
	if 0 == len(issues) {
		return fault.ErrMissingParameters
	}

	_, duplicate, err := reservoir.StoreIssues(issues)
	if nil != err {
		return err
	}

	if duplicate {
		return fault.ErrTransactionAlreadyExists
	}

	return nil
}

// unpack transfer and process it
func processTransfer(packed []byte) error {

	if 0 == len(packed) {
		return fault.ErrMissingParameters
	}

	if !mode.Is(mode.Normal) {
		return fault.ErrNotAvailableDuringSynchronise
	}

	transaction, _, err := transactionrecord.Packed(packed).Unpack(mode.IsTesting())
	if nil != err {
		return err
	}

	transfer, ok := transaction.(transactionrecord.BitmarkTransfer)
	if !ok {
		return fault.ErrTransactionIsNotATransfer
	}

	_, duplicate, err := reservoir.StoreTransfer(transfer)
	if nil != err {
		return err
	}
	if duplicate {
		return fault.ErrTransactionAlreadyExists
	}

	return nil
}

// process proof block
func processProof(packed []byte) error {

	if 0 == len(packed) {
		return fault.ErrMissingParameters
	}

	if !mode.Is(mode.Normal) {
		return fault.ErrNotAvailableDuringSynchronise
	}

	var payId pay.PayId
	if len(packed) > payment.NonceLength+len(payId) {
		return fault.ErrInvalidNonce
	}

	copy(payId[:], packed[:len(payId)])
	nonce := packed[len(payId):]
	status := reservoir.TryProof(payId, nonce)
	if reservoir.TrackingAccepted != status {
		// pay id already processed or was invalid
		return fault.ErrPayIdAlreadyUsed
	}

	return nil
}
