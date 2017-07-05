// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package bitcoin

import (
	"encoding/hex"
	"time"

	"github.com/bitmark-inc/bitmarkd/constants"
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/currency/satoshi"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/logger"
)

const (
	bitcoinConfirmations = 3     // stop processing this many blocks back from most recent block
	maximumBlockRate     = 500.0 // blocks per second
)

// wait for new blocks
func (state *bitcoinData) Run(args interface{}, shutdown <-chan struct{}) {

	log := state.log

loop:
	for {
		log.Debug("waiting…")
		select {
		case <-shutdown:
			break loop

		case <-time.After(60 * time.Second):
			var headers []blockHeader
			err := util.FetchJSON(globalData.client, state.url+"/headers/1/"+state.latestBlockHash+".json", &headers)
			if nil != err {
				log.Errorf("headers: error: %s", err)
				continue loop
			}
			if len(headers) < 1 {
				continue loop
			}

			log.Infof("block number: %d confirmations: %d", headers[0].Height, headers[0].Confirmations)

			if state.forward && headers[0].Confirmations <= bitcoinConfirmations {
				continue loop
			}

			state.process()
		}
	}
}

func (state *bitcoinData) process() {

	log := state.log
	hash := state.latestBlockHash

	startTime := time.Now()
	counter := 0

	// if in reverse only go back this far
	originTime := time.Now().Add(-constants.ReservoirTimeout)

loop:
	for {
		var block block
		err := util.FetchJSON(globalData.client, state.url+"/block/"+hash+".json", &block)
		if nil != err {
			log.Errorf("failed block from hash: %s", hash)
			return
		}

		if block.Confirmations <= bitcoinConfirmations {
			if !state.forward {
				hash = block.PreviousBlockHash
				state.latestBlockHash = hash
				continue loop
			}
			state.latestBlockHash = hash
			break
		}

		log.Infof("block: %d  hash: %q", block.Height, block.Hash)
		log.Tracef("block contents: %#v", block)

		transactionCount := len(block.Tx) // first is the coinbase and can be ignored
		if transactionCount > 1 {
			log.Infof("block: %d  transactions: %d", block.Height, transactionCount)
			for _, tx := range block.Tx[1:] {
				ScanTx(log, &tx)
			}
		}

		counter += 1 // number of blocks processed

		// rate limit
		if counter > 10 {
			timeTaken := time.Since(startTime)
			rate := float64(counter) / timeTaken.Seconds()
			log.Infof("rate: %f", rate)
			if rate > maximumBlockRate {
				log.Infof("exceeds: %f", maximumBlockRate)
				time.Sleep(2 * time.Second)
			}
		}

		// set up to get next block
		if state.forward {
			hash = block.NextBlockHash
		} else {
			blockTime := time.Unix(block.Time, 0)
			if blockTime.Before(originTime) {
				state.forward = true
				break loop
			}
			hash = block.PreviousBlockHash
		}
	}
}

func ScanTx(log *logger.L, tx *Transaction) {
	hexTxID, err := hex.DecodeString(tx.TxID)
	if err != nil {
		log.Errorf("invalid tx id: %s", tx.TxID)
		return
	}

	var payID pay.PayId
	amounts := make(map[string]uint64)
	found := false

	for _, vout := range tx.Vout {
		if len(vout.ScriptPubKey.Hex) == bitcoin_OP_RETURN_RECORD_LENGTH && vout.ScriptPubKey.Hex[0:4] == bitcoin_OP_RETURN_HEX_CODE {
			pid := vout.ScriptPubKey.Hex[bitcoin_OP_RETURN_PAY_ID_OFFSET:]
			if err := payID.UnmarshalText([]byte(pid)); err != nil {
				log.Errorf("invalid pay id: %s", pid)
				return
			}

			found = true
			continue
		}

		if len(vout.ScriptPubKey.Addresses) == 1 {
			amounts[vout.ScriptPubKey.Addresses[0]] += satoshi.FromByteString(vout.Value)
		}
	}

	if !found {
		return
	}

	if len(amounts) == 0 {
		log.Warnf("found pay id but no payments in tx id: %s", tx.TxID)
		return
	}

	log.Infof("store bitcoin tx id: %s for pay id: %s", tx.TxID, payID)
	packed := packPaymentDetails(hexTxID, amounts)
	storage.Pool.Payment.Put(payID[:], packed)
}

func packPaymentDetails(hexTxID []byte, amounts map[string]uint64) []byte {
	// create packed structure to store payment details
	packed := util.ToVarint64(currency.Bitcoin.Uint64())

	// transaction ID
	packed = append(packed, util.ToVarint64(uint64(len(hexTxID)))...)
	packed = append(packed, hexTxID...)

	// number of Vout payments
	packed = append(packed, util.ToVarint64(uint64(len(amounts)))...)

	// individual payments
	for address, value := range amounts {
		packed = append(packed, util.ToVarint64(uint64(len(address)))...)
		packed = append(packed, address...)
		packed = append(packed, util.ToVarint64(value)...)
	}

	return packed
}
