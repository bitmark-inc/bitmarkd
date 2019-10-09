package p2p

import (
	"context"
	"fmt"
	"os"

	"github.com/bitmark-inc/bitmarkd/asset"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/payment"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// SubHandler multicasting subscription handler
func (n *Node) SubHandler(ctx context.Context, sub *pubsub.Subscription) {
	log := n.Log
	log.Info("-- Sub start listen --")
	nodeChain := mode.ChainName()
loop:
	for {
		msg, err := sub.Next(ctx)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue loop
		}
		chain, fn, parameters, err := UnPackP2PMessage(msg.Data)
		if chain != nodeChain {
			log.Errorf("-->>Different Chain Error: this chain %v peer chain %v", nodeChain, chain)
			continue loop
		}
		dataLength := len(parameters)

		switch fn {
		case "block":
			if dataLength < 1 {
				log.Debugf("-->>block with too few data: %d items", dataLength)
				continue loop
			}
			log.Infof("-->>received block: %x", parameters[0])
			if !mode.Is(mode.Normal) {
				err := fault.ErrNotAvailableDuringSynchronise
				log.Debugf("-->>failed assets: error: %s", err)
				continue loop
			} else {
				messagebus.Bus.Blockstore.Send("remote", parameters[0])
			}
		case "assets":
			if dataLength < 1 {
				log.Debugf("-->>assets with too few data: %d items", dataLength)
				continue loop
			}
			log.Infof("-->>received assets: %x", parameters[0])
			err := processAssets(parameters[0])
			if nil != err {
				log.Debugf("-->>failed assets: error: %s", err)
				continue loop
			}
		case "issues":
			if dataLength < 1 {
				log.Debugf("-->>issues with too few data: %d items", dataLength)
				continue loop
			}
			log.Infof("-->>received issues: %x", parameters[0])
			err := processIssues(parameters[0])
			if nil != err {
				log.Debugf("-->>failed issues: error: %s", err)
				continue loop
			}
		case "transfer":
			if dataLength < 1 {
				log.Debugf("-->>transfer with too few data: %d items", dataLength)
				continue loop
			}
			log.Infof("-->>received transfer: %x", parameters[0])
			err := processTransfer(parameters[0])
			if nil != err {
				log.Debugf("-->>failed transfer: error: %s", err)
				continue loop
			}

		case "proof":
			if dataLength < 1 {
				log.Debugf("-->>proof with too few data: %d items", dataLength)
				continue loop
			}
			log.Infof("-->>received proof: %x", parameters[0])
			err := processProof(parameters[0])
			if nil != err {
				log.Debugf("-->>failed proof: error: %s", err)
				continue loop
			}
		case "rpc":
			if dataLength < 3 {
				log.Debugf("-->>rpc with too few data: %d items", dataLength)
				continue loop
			}
			if 8 != len(parameters[2]) {
				log.Debug("-->>rpc with invalid timestamp")
				continue loop
			}
			messagebus.Bus.Announce.Send("addrpc", parameters[0], parameters[1], parameters[2])
		case "peer":
			if dataLength < 3 {
				log.Debugf("-->>peer with too few data: %d items", dataLength)
				continue loop
			}
			if 8 != len(parameters[2]) {
				log.Debugf("-->>peer with invalid timestamp=%v", parameters[2])
				continue loop
			}
			id, err := peer.IDFromBytes(parameters[0])
			log.Infof("\x1b[32m-->>sub Recieve: %v  ID:%s\x1b[0m \n", fn, id.ShortString())
			if err != nil {
				log.Error("\x1b[31m-->>invalid id in requesting\x1b[0m")
				continue loop
			}
			messagebus.Bus.Announce.Send("addpeer", parameters[0], parameters[1], parameters[2])
		default:
			log.Infof("-->>unreganized Command:%s ", fn)
			continue loop
		}
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

	// all items were duplicates
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

	issues := make([]*transactionrecord.BitmarkIssue, 0, reservoir.MaximumIssues)
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

	duplicate := false

	transfer, ok := transaction.(transactionrecord.BitmarkTransfer)
	if ok {

		_, duplicate, err = reservoir.StoreTransfer(transfer)

	} else {
		switch tx := transaction.(type) {

		case *transactionrecord.ShareGrant:
			_, duplicate, err = reservoir.StoreGrant(tx)

		case *transactionrecord.ShareSwap:
			_, duplicate, err = reservoir.StoreSwap(tx)

		default:
			return fault.ErrTransactionIsNotATransfer
		}
	}

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
	nonceLength := len(packed) - len(payId) // could be negative
	if nonceLength < payment.MinimumNonceLength || nonceLength > payment.MaximumNonceLength {
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
