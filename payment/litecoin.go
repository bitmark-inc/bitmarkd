// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package payment

import (
	"encoding/hex"
	"encoding/json"
	"github.com/bitmark-inc/bitmarkd/constants"
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/currency/satoshi"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/logger"
	"net/http"
	"sync"
	"time"
)

const (
	litecoinOPReturnHexCode      = "6a30" // op code with 48 byte parameter
	litecoinOPReturnPrefixLength = len(litecoinOPReturnHexCode)
	litecoinOPReturnPayIDOffset  = litecoinOPReturnPrefixLength
	litecoinOPReturnRecordLength = litecoinOPReturnPrefixLength + 2*48
)

type litecoinScriptPubKey struct {
	Hex       string   `json:"hex"`
	Addresses []string `json:"addresses"`
}

type litecoinVout struct {
	Value        json.RawMessage      `json:"value"`
	ScriptPubKey litecoinScriptPubKey `json:"scriptPubKey"`
}

type litecoinTransaction struct {
	TxId string         `json:"txid"`
	Vout []litecoinVout `json:"vout"`
}

type litecoinBlock struct {
	Hash              string                `json:"hash"`
	Confirmations     uint64                `json:"confirmations"`
	Height            uint64                `json:"height"`
	Tx                []litecoinTransaction `json:"tx"`
	Time              int64                 `json:"time"`
	PreviousBlockHash string                `json:"previousblockhash"`
	NextBlockHash     string                `json:"nextblockhash"`
}

type litecoinBlockHeader struct {
	Hash              string `json:"hash"`
	Confirmations     uint64 `json:"confirmations"`
	Height            uint64 `json:"height"`
	Time              int64  `json:"time"`
	PreviousBlockHash string `json:"previousblockhash"`
	NextBlockHash     string `json:"nextblockhash"`
}

type litecoinChainInfo struct {
	Chain  string `json:"chain"`
	Blocks uint64 `json:"blocks"`
	Hash   string `json:"bestblockhash"`
}

// litecoinHandler implements the currencyHandler interface for litecoin
type litecoinHandler struct {
	log   *logger.L
	state *litecoinState
}

func newLitecoinHandler(useDiscovery bool, conf *currencyConfiguration) (*litecoinHandler, error) {
	log := logger.New("litecoin")

	if useDiscovery {
		return &litecoinHandler{log: log}, nil
	}

	state, err := newLitecoinState(conf.URL)
	if err != nil {
		return nil, err
	}
	return &litecoinHandler{log, state}, nil
}

func (h *litecoinHandler) processPastTxs(dat []byte) {
	txs := make([]litecoinTransaction, 0)
	if err := json.Unmarshal(dat, &txs); err != nil {
		h.log.Errorf("unable to unmarshal txs: %v", err)
		return
	}

	for _, tx := range txs {
		h.log.Debugf("old possible payment tx received: %s\n", tx.TxId)
		inspectLitecoinTx(h.log, &tx)
	}
}

func (h *litecoinHandler) processIncomingTx(dat []byte) {
	var tx litecoinTransaction
	if err := json.Unmarshal(dat, &tx); err != nil {
		h.log.Errorf("unable to unmarshal tx: %v", err)
		return
	}

	h.log.Infof("new possible payment tx received: %s\n", tx.TxId)
	inspectLitecoinTx(h.log, &tx)
}

func (h *litecoinHandler) checkLatestBlock(wg *sync.WaitGroup) {
	defer wg.Done()

	var headers []litecoinBlockHeader
	if err := util.FetchJSON(h.state.client, h.state.url+"/headers/1/"+h.state.latestBlockHash+".json", &headers); err != nil {
		h.log.Errorf("headers: error: %s", err)
		return
	}

	if len(headers) < 1 {
		return
	}

	h.log.Infof("block number: %d confirmations: %d", headers[0].Height, headers[0].Confirmations)

	if h.state.forward && headers[0].Confirmations <= requiredConfirmations {
		return
	}

	h.state.process(h.log)
}

// litecoinState maintains the current block state and extracts possible payment txs from litecoin blocks
type litecoinState struct {
	// connection to litecoin daemon
	client *http.Client
	url    string

	// values from litecoind
	latestBlockNumber uint64
	latestBlockHash   string

	// scanning direction
	forward bool
}

func newLitecoinState(url string) (*litecoinState, error) {
	client := &http.Client{}

	var chain litecoinChainInfo
	if err := util.FetchJSON(client, url+"/chaininfo.json", &chain); err != nil {
		return nil, err
	}

	if !mode.IsTesting() && chain.Chain != "main" {
		return nil, fault.ErrInvalidChain
	}

	return &litecoinState{
		client:            client,
		url:               url,
		latestBlockNumber: chain.Blocks,
		latestBlockHash:   chain.Hash,
		forward:           false,
	}, nil
}

func (state *litecoinState) process(log *logger.L) {
	counter := 0                                                 // number of blocks processed
	startTime := time.Now()                                      // used to calculate the elapsed time of the process
	traceStopTime := time.Now().Add(-constants.ReservoirTimeout) // reverse scan stops when the block is older than traceStopTime

	hash := state.latestBlockHash

process_blocks:
	for {
		var block litecoinBlock
		if err := util.FetchJSON(state.client, state.url+"/block/"+hash+".json", &block); err != nil {
			log.Errorf("failed to get the block by hash: %s", hash)
			return
		}
		log.Infof("height: %d hash: %q number of txs: %d", block.Height, block.Hash, len(block.Tx))
		log.Tracef("block: %#v", block)

		if block.Confirmations <= requiredConfirmations {
			if !state.forward {
				hash = block.PreviousBlockHash
				state.latestBlockHash = hash
				continue process_blocks
			}
			state.latestBlockHash = hash
			break process_blocks
		}

		// extract possible payment txs from the block
		transactionCount := len(block.Tx) // ignore the first tx (coinbase tx)
		if transactionCount > 1 {
			for _, tx := range block.Tx[1:] {
				inspectLitecoinTx(log, &tx)
			}
		}

		// throttle the sync speed
		counter++
		if counter > 10 {
			timeTaken := time.Since(startTime)
			rate := float64(counter) / timeTaken.Seconds()
			if rate > maximumBlockRate {
				log.Infof("the current rate %f exceeds the limit %f", rate, maximumBlockRate)
				time.Sleep(2 * time.Second)
			}
		}

		// move to the next block
		if state.forward {
			hash = block.NextBlockHash
		} else {
			blockTime := time.Unix(block.Time, 0)
			if blockTime.Before(traceStopTime) {
				state.forward = true
				break process_blocks
			}
			hash = block.PreviousBlockHash
		}
	}
}

func inspectLitecoinTx(log *logger.L, tx *litecoinTransaction) {
	_, err := hex.DecodeString(tx.TxId)
	if err != nil {
		log.Errorf("invalid tx id: %s", tx.TxId)
		return
	}

	var payID pay.PayId
	amounts := make(map[string]uint64)
	found := false

scan_vouts:
	for _, vout := range tx.Vout {
		if len(vout.ScriptPubKey.Hex) == litecoinOPReturnRecordLength && vout.ScriptPubKey.Hex[0:4] == litecoinOPReturnHexCode {
			pid := vout.ScriptPubKey.Hex[litecoinOPReturnPayIDOffset:]
			if err := payID.UnmarshalText([]byte(pid)); err != nil {
				log.Errorf("invalid pay id: %s", pid)
				return
			}

			found = true
			continue scan_vouts
		}

		if len(vout.ScriptPubKey.Addresses) == 1 {
			amounts[vout.ScriptPubKey.Addresses[0]] += satoshi.FromByteString(vout.Value)
		}
	}

	if !found {
		return
	}

	if len(amounts) == 0 {
		log.Warnf("found pay id but no payments in tx id: %s", tx.TxId)
		return
	}

	reservoir.SetTransferVerified(
		payID,
		&reservoir.PaymentDetail{
			Currency: currency.Litecoin,
			TxID:     tx.TxId,
			Amounts:  amounts,
		},
	)
}
