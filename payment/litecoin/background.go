// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package litecoin

import (
	"encoding/hex"
	"encoding/json"
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
	litecoinConfirmations = 3     // stop processing this many blocks back from most recent block
	maximumBlockCount     = 10000 // total blocks in one download
	maximumBlockRate      = 500.0 // blocks per second
)

type BlockHeader struct {
	Hash              string `json:"hash"`
	Confirmations     uint64 `json:"confirmations"`
	Height            uint64 `json:"height"`
	Time              int64  `json:"time"`
	PreviousBlockHash string `json:"previousblockhash"`
	NextBlockHash     string `json:"nextblockhash"`
}

type Transaction struct {
	TxId string `json:"txid"`
	//Hash string `json:"hash"`
	//Vin  []Vin  `json:"vin"`
	Vout []Vout `json:"vout"`
}

type Block struct {
	Hash              string        `json:"hash"`
	Confirmations     uint64        `json:"confirmations"`
	Height            uint64        `json:"height"`
	Tx                []Transaction `json:"tx"`
	Time              int64         `json:"time"`
	PreviousBlockHash string        `json:"previousblockhash"`
	NextBlockHash     string        `json:"nextblockhash"`
}

type ScriptPubKey struct {
	Hex       string   `json:"hex"`
	Addresses []string `json:"addresses"`
}

type Vout struct {
	Value        json.RawMessage `json:"value"`
	ScriptPubKey ScriptPubKey    `json:"scriptPubKey"`
}

// wait for new blocks
func (state *litecoinData) Run(args interface{}, shutdown <-chan struct{}) {

	log := state.log

loop:
	for {
		log.Debug("waiting…")
		select {
		case <-shutdown:
			break loop

		case <-time.After(60 * time.Second):
			var headers []BlockHeader
			err := util.FetchJSON(state.client, state.url+"/headers/1/"+state.latestBlockHash+".json", &headers)
			if nil != err {
				log.Errorf("headers: error: %s", err)
				continue loop
			}
			if len(headers) < 1 {
				continue loop
			}

			log.Infof("block number: %d confirmations: %d", headers[0].Height, headers[0].Confirmations)

			if state.forward && headers[0].Confirmations <= litecoinConfirmations {
				continue loop
			}

			state.process()
		}
	}
}

const (
	litecoin_OP_RETURN_HEX_CODE      = "6a30" // op code with 48 byte parameter
	litecoin_OP_RETURN_PREFIX_LENGTH = len(litecoin_OP_RETURN_HEX_CODE)
	litecoin_OP_RETURN_PAY_ID_OFFSET = litecoin_OP_RETURN_PREFIX_LENGTH
	litecoin_OP_RETURN_RECORD_LENGTH = litecoin_OP_RETURN_PREFIX_LENGTH + 2*48
)

func (state *litecoinData) process() {

	log := state.log
	hash := state.latestBlockHash

	startTime := time.Now()
	counter := 0

	// if in reverse only go back this far
	originTime := time.Now().Add(-constants.ReservoirTimeout)

loop:
	for {
		var block Block
		err := util.FetchJSON(state.client, state.url+"/block/"+hash+".json", &block)
		if nil != err {
			log.Errorf("failed block from hash: %s", hash)
			return
		}

		if block.Confirmations <= litecoinConfirmations {
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
				CheckForPaymentTransaction(log, &tx)
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

func CheckForPaymentTransaction(log *logger.L, tx *Transaction) {

	// scan all Vout looking for script with OP_RETURN
	for j, vout := range tx.Vout {
		if litecoin_OP_RETURN_RECORD_LENGTH == len(vout.ScriptPubKey.Hex) && litecoin_OP_RETURN_HEX_CODE == vout.ScriptPubKey.Hex[0:4] {
			var payId pay.PayId
			payIdBytes := []byte(vout.ScriptPubKey.Hex[litecoin_OP_RETURN_PAY_ID_OFFSET:])
			err := payId.UnmarshalText(payIdBytes)
			if nil != err {
				log.Errorf("failed to get pay id error: %s", err)
			} else {
				log.Infof("possible tx id: %s", tx.TxId)
				//log.Debugf("possible transaction: %#v", *tx)
				scanTx(log, payId, j, tx)
			}
			break
		}
	}

}

func scanTx(log *logger.L, payId pay.PayId, payIdIndex int, tx *Transaction) {

	amounts := make(map[string]uint64)

	// extract payments, skipping already determine OP_RETURN vout
	for i, vout := range tx.Vout {
		log.Debugf("vout[%d]: %v ", i, vout)
		if payIdIndex == i {
			continue
		}
		if 1 == len(vout.ScriptPubKey.Addresses) {
			amounts[vout.ScriptPubKey.Addresses[0]] += satoshi.FromByteString(vout.Value)
		}
	}

	if 0 == len(amounts) {
		log.Warnf("found pay id but no payments in tx id: %s", tx.TxId)
		return
	}

	// create packed structure to store payment details
	packed := util.ToVarint64(currency.Litecoin.Uint64())

	// transaction ID
	txId, err := hex.DecodeString(tx.TxId)
	if nil != err {
		log.Errorf("decode litecoin tx id error: %s", err)
		return
	}
	packed = append(packed, util.ToVarint64(uint64(len(txId)))...)
	packed = append(packed, txId...)

	// number of Vout payments
	packed = append(packed, util.ToVarint64(uint64(len(amounts)))...)

	// individual payments
	for address, value := range amounts {
		packed = append(packed, util.ToVarint64(uint64(len(address)))...)
		packed = append(packed, address...)
		packed = append(packed, util.ToVarint64(value)...)
	}

	log.Infof("store litecoin tx id: %s for pay id: %s", tx.TxId, payId)
	storage.Pool.Payment.Put(payId[:], packed)
}
