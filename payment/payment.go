// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package payment

import (
	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/transaction"
	"github.com/bitmark-inc/logger"
	"strings"
	"sync"
	"time"
)

// global constants
const (
	paymentVerifyInterval = 2 * time.Minute // block time of currency with lowest block mining time
	paymentExpiryTime     = 2 * time.Hour   // how long to keep unpaid items
	paymentChunkSize      = 100             // maximum transactions to process in one interval

	maximumAddresses         = 60 // keep addresses from this many blocks (2 minutes/block => 2 hours == 2 * record expiry)
	forkProtection           = 10 // keep this far behind on bitmark block chain
	currencyAddressSeparator = ":"
	panicMessage             = "payment module is not initialised"
)

// for currency specific methods
type callType struct {
	pay   func(paymentData []byte, count int) error
	miner func() string // returns string form of address in that currencies usual encoding
}

// globals for background proccess
type paymentData struct {
	sync.RWMutex // to allow locking

	// logger
	log *logger.L

	// counter to detect when to finalise
	nestingLevel int

	// data pools
	paid map[transaction.Link]struct{}

	// valid miner addresses
	validMiners *circular

	// the current payment address
	currentPaymentAddresses []block.MinerAddress

	// currency APIs
	calls map[string]*callType

	// for background
	background *background.T

	// set once during initialise
	initialised bool
}

// global data
var globalData paymentData

// list of backgrounds to start
var paymentProcesses = background.Processes{
	paymentBitmarkBlockScanner,
	paymentVerifier,
}

// internal APIs - called by currency modules (e.g. bitcoin.go)
// ------------------------------------------------------------

// all payment methods call this
func paymentInitialise() error {
	globalData.Lock()
	defer globalData.Unlock()

	globalData.nestingLevel += 1

	// no need to start if already started
	if globalData.initialised {
		return nil
	}

	globalData.log = logger.New("payment")
	if nil == globalData.log {
		return fault.ErrInvalidLoggerChannel
	}
	globalData.log.Info("starting…")

	// initialise
	globalData.calls = make(map[string]*callType)

	// map of paid transaction ids
	globalData.paid = make(map[transaction.Link]struct{})

	// initialise the circular buffer of miner addresses
	globalData.validMiners = newCircular(maximumAddresses)

	// all data initialised
	globalData.initialised = true

	// start background processes
	globalData.log.Info("start background")
	globalData.background = background.Start(paymentProcesses, globalData.log)

	return nil
}

// finialise - stop all background tasks
func paymentFinalise() error {
	globalData.Lock()
	defer globalData.Unlock()

	if !globalData.initialised {
		return nil
	}

	// check if allnested initialise calls are done
	globalData.nestingLevel -= 1
	if globalData.nestingLevel > 0 {
		return nil
	}

	// final log message
	globalData.log.Info("shutting down…")

	// shutdown all background tasks
	background.Stop(globalData.background)

	// destroy the buffer
	globalData.validMiners.destroy()

	// finally...
	globalData.log.Flush()
	globalData.initialised = false
	return nil
}

// register currency specific operations
// called by currency module
func register(currency string, c *callType) {

	globalData.Lock()
	defer globalData.Unlock()

	if !globalData.initialised {
		fault.Panic(panicMessage)
	}

	globalData.calls[strings.ToLower(currency)] = c
}

// is a particular address a valid miner
// called by currency module
func isMinerAddress(currency string, address string) bool {
	globalData.RLock()
	defer globalData.RUnlock()

	if !globalData.initialised {
		fault.Panic(panicMessage)
	}

	a := block.MinerAddress{
		Currency: currency,
		Address:  address,
	}
	return globalData.validMiners.isPresent(a)
}

// mark a transaction ID as paid
func markPaid(txId transaction.Link) {
	globalData.Lock()
	defer globalData.Unlock()

	if !globalData.initialised {
		fault.Panic(panicMessage)
	}

	globalData.log.Infof("mark paid: %#v", txId)

	globalData.paid[txId] = struct{}{}
}

// mark a transaction IDs as expired - to reclaim memory
//
// Called as part of currency background to expire old payment
// transactions.  That process should make sure that this is called
// with payment transactions thare are approximately twice as old as
// the bitmark tranasction expiry period.
func markExpired(txIds []transaction.Link) {
	globalData.Lock()
	defer globalData.Unlock()

	if !globalData.initialised {
		fault.Panic(panicMessage)
	}

	for _, txId := range txIds {
		delete(globalData.paid, txId)
	}
}

// check if a transaction ID is already paid
func isPaid(txId transaction.Link) bool {
	globalData.RLock()
	defer globalData.RUnlock()

	if !globalData.initialised {
		fault.Panic(panicMessage)
	}

	globalData.log.Debugf("is paid: map: %#v", globalData.paid)

	if _, ok := globalData.paid[txId]; ok {
		globalData.log.Infof("is paid: %#v  status: %v", txId, ok)
		return ok
	}

	// this allows the first few blocks to be free
	// to allow the system to be started
	return nil == globalData.currentPaymentAddresses
}

// external APIs
// -------------

// make a payment - primary payment API
// detects currency and makes payment
func Pay(currency string, paymentData []byte, count int) error {
	globalData.RLock()
	defer globalData.RUnlock()

	if !globalData.initialised {
		fault.Panic(panicMessage)
	}

	c, ok := globalData.calls[strings.ToLower(currency)]
	if !ok {
		return fault.ErrInvalidCurrency
	}

	return c.pay(paymentData, count)
}

// for miner routines to get this nodes addresses
func MinerAddresses() []block.MinerAddress {

	globalData.RLock()
	defer globalData.RUnlock()

	if !globalData.initialised {
		fault.Panic(panicMessage)
	}

	m := make([]block.MinerAddress, len(globalData.calls))
	i := 0
	for currency, call := range globalData.calls {
		m[i].Currency = currency
		m[i].Address = call.miner()
	}

	return m
}

// for RPC to get payment addresses - if any
func PaymentAddresses() []block.MinerAddress {

	globalData.RLock()
	defer globalData.RUnlock()

	if !globalData.initialised {
		fault.Panic(panicMessage)
	}

	return globalData.currentPaymentAddresses
}

// check if paid and set paid flag
//
// returns true on transition from unpaid to paid
func CheckPaid(txId transaction.Link) bool {
	if isPaid(txId) {
		if state, found := txId.State(); found && transaction.PendingTransaction == state {
			txId.SetState(transaction.VerifiedTransaction)
			return true
		}
	}
	return false
}

// background processing
// ---------------------

// scan the bitmark block chain keeping some distance from the highest
// block to provide protection against forks
func paymentBitmarkBlockScanner(args interface{}, shutdown <-chan bool, finished chan<- bool) {

	log := args.(*logger.L)

	currentBlockNumber := uint64(1)
	highestBlockNumber := currentBlockNumber

loop:
	for {
		if block.Number() > forkProtection {
			highestBlockNumber = block.Number() - forkProtection
		}
		if currentBlockNumber >= highestBlockNumber {
			select {
			case <-shutdown:
				break loop
			case <-time.After(2 * time.Minute): // polling limit
			}
			continue loop
		} else {
			select {
			case <-shutdown:
				break loop
			default:
			}
		}
		log.Infof("block: %d", currentBlockNumber)
		packedBlock, ok := block.Get(currentBlockNumber)
		if !ok {
			log.Criticalf("failed to get block: %d", currentBlockNumber)
			fault.Panic("paymentBackground failed to get block")
		}

		var blk block.Block
		if err := packedBlock.Unpack(&blk); nil != err {
			log.Errorf("failed to unpack block: %d  error: %v", currentBlockNumber, err)
			fault.Panic("paymentBackground failed to unpack block")
		}

		storeMinerAddress(blk.Addresses)

		// try next block
		currentBlockNumber += 1
	}

	close(finished)
}

// is a particular address a valid miner
func storeMinerAddress(addresses []block.MinerAddress) {
	globalData.Lock()
	defer globalData.Unlock()

	if !globalData.initialised {
		fault.Panic(panicMessage)
	}

	// check for valid addresses
	n := 0
	for _, a := range addresses {
		if "" != a.Currency {
			n += 1
		}
	}

	// nothing to put
	if 0 == n {
		return
	}

	if len(addresses) == n {

		// update current payment addresses
		globalData.currentPaymentAddresses = addresses

		// add addresses to pool for payment receipt
		globalData.validMiners.put(addresses)

	} else {

		// prepare subset
		subset := make([]block.MinerAddress, 0, n)
		for _, a := range addresses {
			if "" != a.Currency {
				continue
			}
			subset = append(subset, a)
		}

		// update current payment addresses
		globalData.currentPaymentAddresses = subset

		// add addresses to pool for payment receipt
		globalData.validMiners.put(subset)

	}

}

// payment verification
func paymentVerifier(args interface{}, shutdown <-chan bool, finished chan<- bool) {

	log := args.(*logger.L)
	log.Info("verify: starting…")

loop:
	for {
		index := transaction.IndexCursor(0)

		select {
		case <-shutdown:
			break loop

		case <-time.After(paymentVerifyInterval):
		}

		log.Info("verify: process")
	process:
		for {

			select {
			case <-shutdown:
				break loop
			default:
			}

			results := index.FetchUnpaid(paymentChunkSize)
			if len(results) <= 0 {
				break process
			}

			// any unpaid tx older that this will be expired
			expiryTime := time.Now().UTC().Add(-paymentExpiryTime)

			for _, item := range results {
				txId := item.Link
				log.Debugf("verify: check: %#v", txId)
				if CheckPaid(txId) {
					log.Infof("verify: paid: %#v", txId)
					continue
				}

				log.Debugf("verify: time: exp: %v  created: %v", expiryTime, item.Timestamp)
				// not paid so check if expired
				if item.Timestamp.Before(expiryTime) {
					log.Infof("verify: expired: %#v  created: %v", txId, item.Timestamp)
					txId.SetState(transaction.ExpiredTransaction)
				}
			}
		}
	}

	log.Infof("verify: shutting down…")
	close(finished)
}
