// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package mine

import (
	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/payment"
	"github.com/bitmark-inc/bitmarkd/transaction"
	"github.com/bitmark-inc/logger"
	"sync"
	"sync/atomic"
	"time"
)

// constants
const (
	chunkSize           = 500                          // maximum number of transactions to add in one go
	maximumTransactions = 25000                        // maximum number of transactions that can go into a block
	interval            = 10 * time.Second             // rate limit transaction additions
	restartMinutes      = 10 * block.ExpectedMinutes   // if nothing found in this time clear and restart the queue
	restartTimeout      = restartMinutes * time.Minute // … for time.After()
)

// globals for background proccess
var globalBackgroundData struct {
	sync.RWMutex // to allow locking

	// maximum active clients
	maximumConnections int

	// set once during initialise
	initialised bool

	// for logging
	log *logger.L

	// for background processes
	background *background.T
}

// list of background processes to start
var processes = background.Processes{
	assembleBlock,
}

// initialise the background process
func Initialise() error {
	globalBackgroundData.Lock()
	defer globalBackgroundData.Unlock()

	// no need to start if already started
	if globalBackgroundData.initialised {
		return fault.ErrAlreadyInitialised
	}

	globalBackgroundData.log = logger.New("mine-bg")
	globalBackgroundData.log.Info("starting…")

	// set up queue
	initialiseJobQueue()

	globalBackgroundData.initialised = true

	// start background processes
	globalBackgroundData.log.Info("start background")
	globalBackgroundData.background = background.Start(processes, globalBackgroundData.log)

	return nil
}

// finialise - stop all background tasks
func Finalise() error {
	globalBackgroundData.Lock()
	defer globalBackgroundData.Unlock()

	if !globalBackgroundData.initialised {
		return fault.ErrNotInitialised
	}

	background.Stop(globalBackgroundData.background)

	// finally...
	globalBackgroundData.log.Flush()
	globalBackgroundData.initialised = false
	return nil
}

// assemble records for mining
func assembleBlock(args interface{}, shutdown <-chan bool, finished chan<- bool) {

	log := args.(*logger.L)
	log.Info("assemble: starting…")

loop:
	for {
		cursor := transaction.NewAvailableCursor()
		restart := true
		ids := []block.Digest{}
		jobQueue.clear()

		restartPoint := time.Now().Add(restartTimeout)

	assemble:
		for {
			select {
			case <-shutdown:
				break loop

			//case <-restart:       // if a block was mined restart the assembly
			//	break assemble  // assuming the notifier has already move transactions to mined

			case <-time.After(interval): // periodic polling for new transactions
			}

			// do not bother if no miners are connected
			if 0 == atomic.LoadInt64(&globalMinerCount) {
				//log.Info("mine-assemble: waiting for first miner")
				break assemble
			}

			// check not in re-sync
			if mode.IsNot(mode.Normal) {
				log.Info("mine-assemble: waiting re-sync completion")
				break assemble
			}

			// nothing happened within timeout
			if time.Now().After(restartPoint) {
				log.Info("mine-assemble: timeout")
				break assemble
			}

			// detect a restart condition - is there a better way?
			if jobQueue.isClear() && len(ids) > 0 {
				log.Info("mine-assemble: clear detected")
				break assemble
			}

			enqueue := false
			if restart {
				log.Info("assemble: initial ids")
				ids = cursor.FetchAvailable(chunkSize)
				if len(ids) > 0 {
					log.Debugf("assemble: initial count: %d", len(ids))
					restart = false
					enqueue = true
				}
			} else {
				log.Info("mine-assemble: more ids")

				// incrementally gather more transactions
				moreIds := cursor.FetchAvailable(chunkSize)
				enqueue = len(moreIds) > 0
				if enqueue {
					log.Infof("assemble: more ids: %d", len(moreIds))
					ids = append(ids, moreIds...)
					restart = len(ids)+chunkSize > maximumTransactions
				}
			}
			if enqueue {
				addresses := payment.MinerAddresses()
				log.Infof("assemble: new job: ids: %d  addresses: %#v", len(ids), addresses)
				timestamp := time.Now().UTC()
				jobQueue.add(ids, addresses, timestamp)
				restartPoint = timestamp.Add(restartTimeout) // new job so extend timeout
			}
		}
	}

	log.Info("assemble: shutting down…")
	close(finished)
}
