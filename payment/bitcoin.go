// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package payment

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/configuration"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/transaction"
	"github.com/bitmark-inc/logger"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"
)

// global constants
const (
	bitcoinMinimumVersion = 90200           // do not start if bitcoind older than this
	bitcoinRateLimit      = 15.0            // blocks/second
	bitcoinPollingTime    = 2 * time.Minute // sample bitcoin "blockcount" RPC at this interval
	bitcoinMaximumRetries = 10              // panic after this many consecutive errors
	bitcoinCurrencyName   = "bitcoin"       // all lowercase currency string
	bitcoinBlockRange     = 200             // number of blocks to consider as relevant
	bitcoinConfirmations  = 3               // stop processing this many blocks back from most recent block

	// this is how far back in the bitcoin block chain to start when process begins
	bitcoinBlockOffset = bitcoinBlockRange + bitcoinConfirmations
)

// globals for background proccess
type bitcoinData struct {
	sync.RWMutex // to allow locking

	// logger
	log *logger.L

	// connection to bitcoin daemon
	client *http.Client
	url    string

	// authentication
	username string
	password string

	// identifier for the RPC
	id uint64

	// payment info
	minerAddress      string
	fee               uint64 // value in Satoshis avoid float because of rounding errors
	latestBlockNumber uint64

	// for garbage collection
	expire map[uint64][]transaction.Link

	// for background
	background *background.T

	// set once during initialise
	initialised bool
}

// global data
var globalBitcoinData bitcoinData

// list of background processes to start
var bitcoinProcesses = background.Processes{
	bitcoinBackground,
}

// external API
// ------------

// initialise for bitcoin payments
// also calls the internal initialisePayment() and register()
//
// Note fee is a string value and is converted to Satoshis to avoid rounding errors
func BitcoinInitialise(configuration configuration.BitcoinAccess) error {

	// ensure payments are initialised
	if err := paymentInitialise(); nil != err {
		return err
	}

	globalBitcoinData.Lock()
	defer globalBitcoinData.Unlock()

	// no need to start if already started
	if globalBitcoinData.initialised {
		return fault.ErrAlreadyInitialised
	}

	if "" == configuration.Address {
		return fault.ErrPaymentAddressMissing
	}

	globalBitcoinData.log = logger.New(bitcoinCurrencyName)
	if nil == globalBitcoinData.log {
		return fault.ErrInvalidLoggerChannel
	}
	globalBitcoinData.log.Info("starting…")

	globalBitcoinData.id = 0
	globalBitcoinData.username = configuration.Username
	globalBitcoinData.password = configuration.Password
	globalBitcoinData.url = configuration.URL
	globalBitcoinData.minerAddress = configuration.Address
	globalBitcoinData.fee = convertToSatoshi([]byte(configuration.Fee))
	globalBitcoinData.latestBlockNumber = 0
	globalBitcoinData.expire = make(map[uint64][]transaction.Link, bitcoinBlockRange)

	if "" != configuration.Certificate {
		keyPair, err := tls.LoadX509KeyPair(configuration.Certificate, configuration.PrivateKey)
		if nil != err {
			return err
		}

		certificatePool := x509.NewCertPool()

		data, err := ioutil.ReadFile(configuration.CACertificate)
		if err != nil {
			globalBitcoinData.log.Criticalf("failed to parse certificate from: %q", configuration.CACertificate)
			return err
		}

		if !certificatePool.AppendCertsFromPEM(data) {
			globalBitcoinData.log.Criticalf("failed to parse certificate from: %q", configuration.CACertificate)
			return err
		}

		tlsConfiguration := &tls.Config{
			Certificates:             []tls.Certificate{keyPair},
			RootCAs:                  certificatePool,
			InsecureSkipVerify:       false,
			CipherSuites:             nil,
			PreferServerCipherSuites: true,
			MinVersion:               12, // force 1.2 and above
			MaxVersion:               0,  // no maximum
			CurvePreferences:         nil,
		}

		globalBitcoinData.client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsConfiguration,
			},
		}
	} else {
		globalBitcoinData.client = &http.Client{}
	}

	// all data initialised
	globalBitcoinData.initialised = true

	globalBitcoinData.log.Debug("getinfo…")

	// query bitcoind for status
	// only need to have necessary fields as JSON unmarshaller will igtnore excess
	var reply struct {
		Version uint64 `json:"version"`
		Blocks  uint64 `json:"blocks"`
	}
	err := bitcoinCall("getinfo", []interface{}{}, &reply)
	if nil != err {
		return err
	}

	// check version is sufficient
	if reply.Version < bitcoinMinimumVersion {
		globalBitcoinData.log.Errorf("Bitcoin version: %d < allowed: %d", reply.Version, bitcoinMinimumVersion)
		return fault.ErrInvalidVersion
	} else {
		globalBitcoinData.log.Infof("Bitcoin version: %d", reply.Version)
	}

	// set up current block number
	globalBitcoinData.latestBlockNumber = reply.Blocks
	globalBitcoinData.log.Debugf("block count: %d", globalBitcoinData.latestBlockNumber)

	// start background processes
	globalBitcoinData.log.Info("start background…")
	globalBitcoinData.background = background.Start(bitcoinProcesses, globalBitcoinData.log)

	register(bitcoinCurrencyName, &callType{
		pay:   bitcoinPay,
		miner: bitcoinAddress,
	})

	return nil
}

// finialise - stop all background tasks
// also calls the internal finalisePayment()
func BitcoinFinalise() error {
	globalBitcoinData.Lock()
	defer globalBitcoinData.Unlock()

	if !globalBitcoinData.initialised {
		return fault.ErrNotInitialised
	}

	globalBitcoinData.log.Info("shutting down…")
	globalBitcoinData.log.Flush()

	// stop background
	background.Stop(globalBitcoinData.background)

	// finally...
	globalBitcoinData.initialised = false

	// finalise the main subsystem
	return paymentFinalise()
}

// transaction calls to bitcoind
// -----------------------------

type bitcoinScriptPubKey struct {
	Hex       string   `json:"hex"`
	Addresses []string `json:"addresses"`
}

type bitcoinVout struct {
	Value        json.RawMessage     `json:"value"`
	ScriptPubKey bitcoinScriptPubKey `json:"scriptPubKey"`
}

type bitcoinTransaction struct {
	Vout []bitcoinVout `json:"vout"`
}

// fetch transaction and decode
func bitcoinGetRawTransaction(hash string, reply *bitcoinTransaction) error {
	globalBitcoinData.Lock()
	defer globalBitcoinData.Unlock()

	if !globalBitcoinData.initialised {
		return fault.ErrNotInitialised
	}

	arguments := []interface{}{
		hash,
		1,
	}
	return bitcoinCall("getrawtransaction", arguments, reply)
}

// decode an existing binary transaction
func bitcoinDecodeRawTransaction(tx []byte, reply *bitcoinTransaction) error {
	globalBitcoinData.Lock()
	defer globalBitcoinData.Unlock()

	if !globalBitcoinData.initialised {
		return fault.ErrNotInitialised
	}

	// need to be in hex for bitcoind
	arguments := []interface{}{
		hex.EncodeToString(tx),
	}
	return bitcoinCall("decoderawtransaction", arguments, reply)
}

// send a raw binary transaction
func bitcoinSendRawTransaction(tx []byte, reply *string) error {
	globalBitcoinData.Lock()
	defer globalBitcoinData.Unlock()

	if !globalBitcoinData.initialised {
		return fault.ErrNotInitialised
	}

	// need to be in hex for bitcoind
	arguments := []interface{}{
		hex.EncodeToString(tx),
	}
	return bitcoinCall("sendrawtransaction", arguments, reply)
}

// for mining
// ----------

// to get the current address as string for mining
func bitcoinAddress() string {
	globalBitcoinData.RLock()
	defer globalBitcoinData.RUnlock()

	return globalBitcoinData.minerAddress
}

// Payment confirmation functions
// ------------------------------

// make a payment for some bitmark transactions
func bitcoinPay(payment []byte, count int) (string, error) {

	var reply bitcoinTransaction
	if err := bitcoinDecodeRawTransaction(payment, &reply); nil != err {
		return "", err
	}
	links, addresses, ok := bitcoinValidateTransaction(&reply)
	if !ok {
		return "", fault.ErrInsufficientPayment
	}

	if 0 == len(links) {
		return "", fault.ErrNotABitmarkPayment
	}

	if 0 == len(addresses) {
		return "", fault.ErrNoPaymentToMiner
	}

	if count > 0 && count != len(links) {
		return "", fault.ErrInvalidCount
	}

	var btcId string
	return btcId, bitcoinSendRawTransaction(payment, &btcId)
}

// validate and extract data from a decoded bitcoin transaction
func bitcoinValidateTransaction(tx *bitcoinTransaction) ([]transaction.Link, []string, bool) {

	transactionCount := len(tx.Vout)
	if transactionCount < 1 {
		return nil, nil, false
	}

	txIds := make([]transaction.Link, transactionCount)
	idIndex := 0

	minerAddresses := make([]string, 0, transactionCount)

	total := uint64(0)

	globalBitcoinData.log.Debugf("len vout: %d", len(tx.Vout))

	for i, vout := range tx.Vout {

		amount := convertToSatoshi(vout.Value)
		globalBitcoinData.log.Tracef("vout[%d]: satoshi: %d  data: %v", i, amount, vout)

		if 0 == amount && len(vout.ScriptPubKey.Hex) > 4 {
			script := vout.ScriptPubKey.Hex
			if "6a24" == script[0:4] {
				// counted "OP_RETURN count=36 txid"
				err := transaction.LinkFromHexString(&txIds[idIndex], script[4:])
				if nil != err {
					continue
				}
				globalBitcoinData.log.Tracef("vout[%d]: link[%d]: %#v", i, idIndex, txIds[idIndex])
				idIndex += 1
			} else if "6a" == script[0:2] {
				// uncounted "OP_RETURN txid"
				err := transaction.LinkFromHexString(&txIds[idIndex], script[2:])
				if nil != err {
					continue
				}
				globalBitcoinData.log.Tracef("vout[%d]: link[%d]: %#v", i, idIndex, txIds[idIndex])
				idIndex += 1
			}
			continue
		}

		// see if out has one address it is a valid miner
		addresses := vout.ScriptPubKey.Addresses
		if 1 != len(addresses) {
			continue
		}

		theAddress := addresses[0]

		// ***** FIX THIS: need to figure out how to bootstrap *****
		// currently just allow my own address as valid
		//if isMinerAddress(bitcoinCurrencyName, theAddress) {
		if isMinerAddress(bitcoinCurrencyName, theAddress) || bitcoinAddress() == theAddress {
			globalBitcoinData.log.Tracef("vout[%d]: miner address: %s -> BTC %d", i, theAddress, amount)
			minerAddresses = append(minerAddresses, theAddress)
			total += amount
		}
	}

	// check sufficient fee
	expectedFee := globalBitcoinData.fee * uint64(idIndex)
	feeOk := total >= expectedFee

	globalBitcoinData.log.Debugf("total:  BTC %d  expected:  BTC %d  ok: %v", total, expectedFee, feeOk)

	return txIds[0:idIndex], minerAddresses, feeOk
}

// low level RPC
// -------------

// high level call - only use while global data locked
// because the HTTP RPC cannot interleave calls and responses
func bitcoinCall(method string, params []interface{}, reply interface{}) error {
	if !globalBitcoinData.initialised {
		fault.Panic("bitcoin not initialised")
	}

	globalBitcoinData.id += 1

	arguments := bitcoinArguments{
		Id:     globalBitcoinData.id,
		Method: method,
		Params: params,
	}
	response := bitcoinReply{
		Result: reply,
	}
	err := bitcoinRPC(&arguments, &response)
	if nil != err {
		globalBitcoinData.log.Tracef("rpc returned error: %v", err)
		return err
	}

	if nil != response.Error {
		s := response.Error.Message
		return fault.ProcessError("Bitcoin RPC error: " + s)
	}
	return nil
}

// for encoding the RPC arguments
type bitcoinArguments struct {
	Id     uint64        `json:"id"`
	Method string        `json:"method"`
	Params []interface{} `json:"params"`
}

// the RPC error response
type bitcoinRpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// for decoding the RPC reply
type bitcoinReply struct {
	Id     int64            `json:"id"`
	Method string           `json:"method"`
	Result interface{}      `json:"result"`
	Error  *bitcoinRpcError `json:"error"`
}

// basic RPC - only use while global data locked
func bitcoinRPC(arguments *bitcoinArguments, reply *bitcoinReply) error {

	s, err := json.Marshal(arguments)
	if nil != err {
		return err
	}

	globalBitcoinData.log.Tracef("rpc send: %s", s)

	postData := bytes.NewBuffer(s)

	request, err := http.NewRequest("POST", globalBitcoinData.url, postData)
	if nil != err {
		return err
	}
	request.SetBasicAuth(globalBitcoinData.username, globalBitcoinData.password)

	response, err := globalBitcoinData.client.Do(request)
	if nil != err {
		return err
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if nil != err {
		return err
	}

	globalBitcoinData.log.Tracef("rpc response body: %s", body)

	err = json.Unmarshal(body, &reply)
	if nil != err {
		return err
	}

	globalBitcoinData.log.Debugf("rpc receive: %s", body)

	return nil
}

// background block reader
// -----------------------

func bitcoinScanBlock(number uint64) error {

	log := globalBitcoinData.log

	transactions, err := bitcoinGetBlock(number)

	// var hash string
	// err := bitcoinCall("getblockhash", []interface{}{number}, &hash)
	// if nil != err {
	// 	return err
	// }

	// log.Debugf("blk %d hash: %s", number, hash)

	// var blk struct {
	// 	Tx []string `json:"tx"`
	// }
	// err = bitcoinCall("getblock", []interface{}{hash}, &blk)
	// if nil != err {
	// 	return err
	// }

	// log.Debugf("blk %d data: %v", number, blk)

	if len(transactions) < 1 {
		log.Debugf("blk %d no transactions", number)
		return nil
	}

	for i, tx := range transactions {
		log.Debugf("blk %d tx %d id: %s", number, i, tx)

		var reply bitcoinTransaction
		err = bitcoinGetRawTransaction(tx, &reply)
		if nil != err {
			continue
		}
		log.Debugf("  tx data: %v", reply)

		// validate transactiona and extract paid items
		links, miners, ok := bitcoinValidateTransaction(&reply)
		if !ok || len(links) < 1 {
			continue
		}
		log.Debugf("  links: %#v  miners: %#v", links, miners)

		// save for expiry
		globalBitcoinData.expire[number] = links

		// mark each ID as paid
		for _, txId := range links {
			markPaid(txId) // ***** Require miners to be stored? ***
		}

	}

	return nil
}

// background to fetch blocks and verify them
// and save info about paid transactions
func bitcoinBackground(args interface{}, shutdown <-chan bool, finished chan<- bool) {

	log := args.(*logger.L)

	// set up the starting block number
	currentBlockNumber := uint64(1)
	if currentBlockNumber > bitcoinBlockOffset {
		currentBlockNumber = globalBitcoinData.latestBlockNumber - bitcoinBlockOffset
	}

loop:
	for {
		// initialise block reading rate limiter
		startTime := time.Now()
		blockCount := 0
		retries := 0

	reading:
		for {
			// compute block rate
			blockCount += 1
			rate := float64(blockCount) / time.Since(startTime).Seconds()

			if rate > bitcoinRateLimit {
				log.Info("reading: waiting…")
				select {
				case <-shutdown:
					break loop
				case <-time.After(time.Second): // rate limit
				}
			} else {
				select {
				case <-shutdown:
					break loop
				default:
				}
			}

			log.Infof("reading: process block: %d", currentBlockNumber)

			if err := bitcoinScanBlock(currentBlockNumber); nil != err {

				log.Infof("  error: %v", err)
				if strings.Contains(err.Error(), "Block height out of range") {
					break reading
				}

				log.Errorf("failed to process block: %d  error: %v", currentBlockNumber, err)
				retries += 1
				if retries > bitcoinMaximumRetries {

					// ***** FIX THIS: need to retry / reset bitcoin RPC connection *****
					fault.Panic("bitcoinBackground maximum retries exceeded")
				}
				continue reading
			}

			retries = 0 // reset if a successful read occurred

			// increment count and go into polling mode if reached confirmation level
			currentBlockNumber += 1
			if currentBlockNumber+bitcoinConfirmations-1 > globalBitcoinData.latestBlockNumber {
				log.Debug("block: set polling")
				break reading
			}

			// expire old payments records (garbage collection)
			if currentBlockNumber > bitcoinBlockOffset {
				n := currentBlockNumber - bitcoinBlockOffset
				if txIds, ok := globalBitcoinData.expire[n]; ok {
					markExpired(txIds)
					delete(globalBitcoinData.expire, n)
				}
			}
		}

		// poll until a new block or blocks are received
	polling:
		for {
			log.Info("polling: waiting…")
			select {
			case <-shutdown:
				break loop
			case <-time.After(bitcoinPollingTime):
			}

			log.Info("polling: process")

			// update the current block number
			n := bitcoinLatestBlockNumber()

			// not enough confirmations - continue polling
			if currentBlockNumber+bitcoinConfirmations <= n {
				log.Debug("block: set reading")
				break polling
			}
		}
	}

	close(finished)
}

// get the transactions in a specific bitcoin block
func bitcoinGetBlock(number uint64) ([]string, error) {
	globalBitcoinData.Lock()
	defer globalBitcoinData.Unlock()

	log := globalBitcoinData.log

	var hash string
	err := bitcoinCall("getblockhash", []interface{}{number}, &hash)
	if nil != err {
		return nil, err
	}

	log.Debugf("blk %d hash: %s", number, hash)

	var blk struct {
		Tx []string `json:"tx"`
	}
	err = bitcoinCall("getblock", []interface{}{hash}, &blk)
	if nil != err {
		return nil, err
	}

	log.Debugf("blk %d data: %v", number, blk)

	return blk.Tx, nil
}

// update and return the current block number
func bitcoinLatestBlockNumber() uint64 {
	globalBitcoinData.Lock()
	defer globalBitcoinData.Unlock()

	var n uint64
	err := bitcoinCall("getblockcount", []interface{}{}, &n)
	if nil == err {
		globalBitcoinData.latestBlockNumber = n
	}

	return globalBitcoinData.latestBlockNumber
}

// convert a string to a Satoshi value
//
// i.e. "0.00000001" will convert to uint64(1)
//
// Note: Invalid characters are simply ignored and the conversion
//       simply stops after 8 decimal places have been processed.
//       Extra decimal points will also be ignored.
func convertToSatoshi(btc []byte) uint64 {

	s := uint64(0)
	point := false
	decimals := 0
	for _, b := range btc {
		if b >= '0' && b <= '9' {
			s *= 10
			s += uint64(b - '0')
			if point {
				decimals += 1
				if decimals >= 8 {
					break
				}
			}
		} else if '.' == b {
			point = true
		}
	}
	for decimals < 8 {
		s *= 10
		decimals += 1
	}

	return s
}
