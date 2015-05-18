// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package peer

import (
	"github.com/bitmark-inc/bilateralrpc"
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/transaction"
	"time"
)

type BlockGetResult struct {
	From  string
	Reply BlockGetReply
	Err   error
}

const (
	retryCount     = 5    // attempts to retrieve block befor failing
	blockRateLimit = 5.0  // maximum blocks per second to fetch
	txRateLimit    = 15.0 // maximum transactions per second to put/get
	txBatchSize    = 10   // number of transactions to fetch from database per loop
)

// resync process
func (t *thread) resynchronise(server *bilateralrpc.Bilateral, n uint64, highest uint64, to []string) {

	log := t.log
	log.Info("resynchronise start")

	retries := 0
	success := true

	startTime := time.Now()
	blockCount := 0

loop:
	for {
		if !success {
			retries += 1
			if retries > retryCount {
				break loop
			}
		} else {
			retries = 0
		}

		// compute block rate
		blockCount += 1
		rate := float64(blockCount) / time.Since(startTime).Seconds()

		if rate > blockRateLimit {
			select {
			case <-t.stop:
				break loop
			case <-time.After(time.Second): // rate limit
			}
		} else {
			select {
			case <-t.stop:
				break loop
			default:
			}
		}

		log.Infof("resynchronisation get block: %d", n)

		// assume failed
		success = false

		args := BlockGetArguments{
			Number: n,
		}
		var result []BlockGetResult
		if err := server.Call(to, "Block.Get", args, &result, 0); nil != err {
			log.Errorf("Block.Get: error: %v", err)
			if n >= highest {
				break loop
			}
			continue loop
		}

		if 0 == len(result) {
			log.Errorf("Block.Get: no reply from: %q", to)
			if n >= highest {
				break loop
			}
			continue loop
		}
		if nil != result[0].Err {
			log.Infof("Block.Get error: %v", result[0].Err)
			break loop
		}

		// this outputs a lot of data…
		log.Tracef("result: %v", result)

		// validate
		packedBlock := block.Packed(result[0].Reply.Data)

		var blk block.Block
		err := packedBlock.Unpack(&blk)
		if nil != err {
			log.Errorf("bad result: %v", result)
			log.Errorf("received block from: %q  error: %v", to, err)
			continue loop
		}

		log.Infof("block: %d  digest: %#v", blk.Number, blk.Digest)

		// fetch the previous block from local storage and see if needs to be replaced
		previousPackedBlock, found := block.Read(n - 1)
		if !found {
			log.Errorf("missing previous block: %d", n-1)
			n -= 1
			continue loop
		}
		var previousBlock block.Block
		err = previousPackedBlock.Unpack(&previousBlock)
		if nil != err {
			log.Errorf("faulty local previous block: error: %v", err)
			n -= 1 // just try to fetch again
			continue loop
		}

		if previousBlock.Digest != blk.Header.PreviousBlock {
			log.Infof("fork detected: digest: %#v  expected: %#v", blk.Header.PreviousBlock, previousBlock.Digest)
			n -= 1
			continue loop
		}

		// get transactions and mark as mined
		if !t.fetchAndMarkAssociatedTransactions(server, &blk, to) {
			log.Errorf("missed some transactions from: %q", to)
			continue loop
		}

		// save block
		packedBlock.Save(n, &blk.Digest)

		// success want next block
		success = true
		n += 1
	}

	log.Info("resynchronisation complete")
}

// for getting transactions
type TransactionGetResult struct {
	From  string
	Reply TransactionGetReply
	Err   error
}

// get all transactions from a block and mark as mined
func (t *thread) fetchAndMarkAssociatedTransactions(server *bilateralrpc.Bilateral, blk *block.Block, addresses []string) bool {

	log := t.log

	log.Info("fetch tx start")

	success := true
	startTime := time.Now()
	txCount := 0

loop:
	for _, txDigest := range blk.TxIds {

		// compute tx rate
		txCount += 1
		rate := float64(txCount) / time.Since(startTime).Seconds()
		log.Debugf("rate: %f  limit: %f", rate, txRateLimit)

		if rate > txRateLimit {
			select {
			case <-t.stop:
				break loop
			case <-time.After(time.Second): // rate limit
			}
		} else {
			select {
			case <-t.stop:
				break loop
			default:
			}
		}

		txid := transaction.Link(txDigest)

		// skip transactions already on file
		state, found := txid.State()
		if found {
			// ***** FIX THIS: possibly need better transaction state machine *****
			if transaction.WaitingIssueTransaction == state {
				txid.SetState(transaction.MinedTransaction)
			} else if transaction.MinedTransaction != state {
				txid.SetState(transaction.AvailableTransaction)
				txid.SetState(transaction.MinedTransaction)
			}
			continue
		}

		// fetch the transaction
		args := TransactionGetArguments{
			TxId: txid,
		}

		// fetch from just from one peer from the list
	fetchOne:
		for _, to := range addresses {
			var result []TransactionGetResult
			if err := server.Call([]string{to}, "Transaction.Get", args, &result, 0); nil != err {
				log.Errorf("Transaction.Get: error: %v", err)
				success = false
				continue fetchOne
			}

			if 0 == len(result) {
				log.Errorf("Transaction.Get: no reply from: %q", to)
				success = false
				continue fetchOne
			}

			// validate
			packedTransaction := transaction.Packed(result[0].Reply.Data)

			_, err := packedTransaction.Unpack()
			if nil != err {
				log.Errorf("received transaction from: %q  error: %v", to, err)
				success = false
				continue fetchOne
			}

			// write the transaction
			log.Infof("txid: %#v", txid)
			txid2, _ := packedTransaction.Write()
			if txid != txid2 {
				log.Errorf("txid: %#v changed to: %#v", txid, txid2)
				success = false
				continue fetchOne
			}

			// got a valid tx - flag as mined
			state, found := txid.State()
			if !found {
				log.Errorf("missing transaction: %#v", txid)
				success = false
				continue fetchOne
			}

			// ***** FIX THIS: possibly need better transaction state machine *****
			if transaction.WaitingIssueTransaction == state {
				txid.SetState(transaction.MinedTransaction)
			} else if transaction.MinedTransaction != state {
				txid.SetState(transaction.AvailableTransaction)
				txid.SetState(transaction.MinedTransaction)
			}

			// transaction sucessfully processed
			success = true
			break fetchOne
		}

	}

	log.Info("fetch tx complete")
	return success
}

// for putting transactions
type TransactionPutResult struct {
	From  string
	Reply TransactionPutReply
	Err   error
}

// rebroadcast all available transactions
func (t *thread) rebroadcastTransactions(server *bilateralrpc.Bilateral) {

	log := t.log

	log.Info("rebroadcast tx start")

	cursor := transaction.NewAvailableCursor()

	startTime := time.Now()
	rate := 0.0
	txCount := 0

loop:
	for {

		if rate > txRateLimit {
			select {
			case <-t.stop:
				break loop
			case <-time.After(time.Second): // rate limit
			}
		} else {
			select {
			case <-t.stop:
				break loop
			default:
			}
		}

		txIds := cursor.FetchAvailable(txBatchSize)
		if 0 == len(txIds) {
			break loop
		}
		log.Infof("rebroadcast count: %d", len(txIds))

		for i, txId := range txIds {

			select {
			case <-t.stop:
				break loop
			default:
			}

			state, packedTx, found := transaction.Link(txId).Read()

			if !found || transaction.AvailableTransaction != state {
				continue
			}

			args := TransactionPutArguments{
				Tx: packedTx,
			}

			log.Infof("rebroadcast [%d/%d] TxId: %#v", i, len(txIds), txId)

			if err := server.Cast(bilateralrpc.SendToAll, "Transaction.Put", args); nil != err {
				log.Errorf("Transaction.Put: error: %v", err)
			}
		}

		// compute tx rate
		txCount += len(txIds)
		rate = float64(txCount) / time.Since(startTime).Seconds()
	}

	log.Info("rebroadcast tx complete")
}
