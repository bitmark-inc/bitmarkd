// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package peer

import (
	"github.com/bitmark-inc/bilateralrpc"
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/payment"
	"github.com/bitmark-inc/bitmarkd/transaction"
)

// loop to read queue
func (peer *peerData) responder(t *thread) {

	t.log.Info("starting…")

	queue := messagebus.Chan()
loop:
	for {
		select {
		case item := <-queue:
			go t.processItem(peer.server, item)

		case <-t.stop:
			break loop
		}
	}

	t.log.Info("shutting down…")
	t.log.Flush()

	close(t.done)
}

// type to hold the broadcast block
type BlockPair struct {
	unpacked block.Block
	packed   block.Packed
}

// process an item
func (t *thread) processItem(server *bilateralrpc.Bilateral, item interface{}) {
	log := t.log

decode:
	switch item.(type) {
	//case int:
	// 	localData.pollNeighbours -= 1
	// 	if localData.pollNeighbours <= 0 {
	// 		localData.pollNeighbours = pollNeighboursTicks
	// 		change, err := pollNeighbours(localData)
	// 		if nil != err {
	// 			return err
	// 		}
	// 		if change {
	// 			if announce.TypePeer == localData.listenType {
	// 				localData.listenType = announce.TypeRPC
	// 			} else {
	// 				localData.listenType = announce.TypeRPC
	// 			}
	// 		}
	// 	} else {
	// 		err := sendPeer(localData)

	// 		if fault.IsErrNotFound(err) {
	// 			return alive(log, peer)
	// 		}
	// 		return err
	// 	}

	case BlockPair: // a block sent from a connected peer
		pair := item.(BlockPair)
		// see  if block number is useful
		if pair.unpacked.Number < block.Number() {
			log.Infof("ignore block: %d\n", pair.unpacked.Number)
			break decode
		}

		// if matching "next" block save, otherwise ignore
		if pair.unpacked.Number != block.Number() || pair.unpacked.Header.PreviousBlock != block.PreviousLink() {
			log.Infof("ignore non-next block: %d\n", pair.unpacked.Number)
			// ignore blocks too far ahead or fork occured
			// (previous digests mismatch), then rely on
			// synchronise to catch up.  Do not forward
			// these blocks to other bitmarkds since we
			// cannot verify the chain integrity.
			break decode
		}

		// ensure have all transactions
		active := server.ActiveConnections()
		if !t.fetchAndMarkAssociatedTransactions(server, &pair.unpacked, active) {
			log.Errorf("missed some transactions from: %q", active)
			break decode  // cannot continue
		}

		// save block only if sucessfully obtained all transactions
		log.Infof("save block: %d\n", pair.unpacked.Number)
		pair.packed.Save(pair.unpacked.Number, &pair.unpacked.Digest)

		// send to everyone else - now local data is all saved
		blockArguments := BlockPutArguments{
			Block: []byte(pair.packed),
		}
		var blockResult []struct {
			From  string
			Reply BlockPutReply
			Err   error
		}
		if err := server.Call(bilateralrpc.SendToAll, "Block.Put", &blockArguments, &blockResult, 0); nil != err {
			// if remote does not accept it is not really a problem for this node - just warn
			log.Warnf("Block.Put err = %v", err)
		} else {
			log.Infof("Block.Put = %v", blockResult)
		}

	case block.Mined: // block created by local miner thread
		packedBlock := item.(block.Mined)
		log.Infof("incoming block.Mined = %x...", packedBlock[:32]) // only shows first 32 bytes

		// our block, so send right away (since we must have all tx already saved)
		blockArguments := BlockPutArguments{
			Block: []byte(packedBlock),
		}
		var blockResult []struct {
			From  string
			Reply BlockPutReply
			Err   error
		}
		if err := server.Call(bilateralrpc.SendToAll, "Block.Put", &blockArguments, &blockResult, 0); nil != err {
			// if remote does not accept it is not really a problem for this node - just warn
			log.Warnf("Block.Put err = %v", err)
		} else {
			log.Infof("Block.Put = %v", blockResult)
		}

	case transaction.Packed: // any incoming Tx either from peers or client RPC
		log.Debugf("incoming Transaction.Packed = %x", item)

		// save record
		txId, justAdded := item.(transaction.Packed).Write()
		log.Infof("incoming TxId = %#v  new = %v", txId, justAdded)

		// send out if this was new
		if justAdded {

			// set paid immediately if possible
			payment.CheckPaid(txId)

			transactionArguments := TransactionPutArguments{
				Tx: item.(transaction.Packed),
			}
			var transactionResult []struct {
				From  string
				Reply TransactionPutReply
				Err   error
			}
			log.Debugf("put TxId: %#v", txId)

			if err := server.Call(bilateralrpc.SendToAll, "Transaction.Put", &transactionArguments, &transactionResult, 0); nil != err {
				// if remote does not accept it is not really a problem for this node - just warn
				log.Warnf("Transaction.Put err = %v", err)
			} else {
				log.Infof("Transaction.Put = %v", transactionResult)
			}
		}

	default:
		log.Errorf("Spurious message: %v", item)
	}
}
