// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package peer

import (
	"bytes"
	"encoding/hex"
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/genesis"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/peer/upstream"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/logger"
	"time"
)

// various timeouts
const (
	cycleInterval       = 15 * time.Second // pause to limit bandwidth
	connectorTimeout    = 60 * time.Second // time out for connections
	samplelingLimit     = 10               // number of cycles to be 1 block out of sync before resync
	fetchBlocksPerCycle = 100              // number of blocks to fetch in one set
	forkProtection      = 10               // fail to fork if height difference is greater than this
	minimumClients      = 3                // do not proceed unless this many clients are connected
)

// a state type for the thread
type connectorState int

// state of the connector process
const (
	cStateConnecting   connectorState = iota // register to nodes and make outgoing connections
	cStateHighestBlock connectorState = iota // locate node(s) with highest block number
	cStateForkDetect   connectorState = iota // read block hashes to check for possible fork
	cStateFetchBlocks  connectorState = iota // fetch blocks from current or fork point
	cStateRebuild      connectorState = iota // rebuild database from fork point (config setting to force total rebuild)
	cStateSampling     connectorState = iota // signal resync complete and sample nodes to see if out of sync occurs
)

type connector struct {
	log          *logger.L
	clients      []*upstream.Upstream
	dynamicStart int
	state        connectorState

	theClient        *upstream.Upstream
	startBlockNumber uint64 // block number where local chain forks
	height           uint64 // block number on best node
	samples          int    // counter to detect missed block broadcast
}

// initialise the connector
func (conn *connector) initialise(privateKey []byte, publicKey []byte, connect []Connection, dynamicEnabled bool) error {

	log := logger.New("connector")
	if nil == log {
		return fault.ErrInvalidLoggerChannel
	}
	conn.log = log

	log.Info("initialising…")

	// allocate all sockets
	staticCount := len(connect) // can be zero
	if 0 == staticCount && !dynamicEnabled {
		log.Error("zero static connections and dynamic is disabled")
		return fault.ErrNoConnectionsAvailable
	}
	conn.clients = make([]*upstream.Upstream, staticCount+offsetCount)
	conn.dynamicStart = staticCount // index of first dynamic socket
	globalData.connectorClients = conn.clients

	// error code for goto fail
	errX := error(nil)

	// initially connect all static sockets
	for i, c := range connect {
		address, err := util.NewConnection(c.Address)
		if nil != err {
			log.Errorf("client[%d]=address: %q  error: %v", i, c.Address, err)
			errX = err
			goto fail
		}
		serverPublicKey, err := hex.DecodeString(c.PublicKey)
		if nil != err {
			log.Errorf("client[%d]=public: %q  error: %v", i, c.PublicKey, err)
			errX = err
			goto fail
		}

		// prevent connection to self
		if bytes.Equal(publicKey, serverPublicKey) {
			errX = fault.ErrConnectingToSelfForbidden
			log.Errorf("client[%d]=public: %q  error: %v", i, c.PublicKey, errX)
			goto fail
		}

		client, err := upstream.New(privateKey, publicKey, connectorTimeout)
		if nil != err {
			log.Errorf("client[%d]=%q  error: %v", i, address, err)
			errX = err
			goto fail
		}

		conn.clients[i] = client

		err = client.Connect(address, serverPublicKey)
		if nil != err {
			log.Errorf("connect[%d]=%q  error: %v", i, address, err)
			errX = err
			goto fail
		}
		log.Infof("public key: %x  at: %q", serverPublicKey, c.Address)
	}

	// just create sockets for dynamic clients
	for i := conn.dynamicStart; i < len(conn.clients); i += 1 {
		client, err := upstream.New(privateKey, publicKey, connectorTimeout)
		if nil != err {
			log.Errorf("client[%d]  error: %v", i, err)
			errX = err
			goto fail
		}

		conn.clients[i] = client
	}

	// start state machine
	conn.state = cStateConnecting

	return nil

	// error handling
fail:
	for _, client := range conn.clients {
		client.Destroy()
	}
	return errX
}

// various RPC calls to upstream connections
func (conn *connector) Run(args interface{}, shutdown <-chan struct{}) {

	log := conn.log

	log.Info("starting…")

	queue := messagebus.Bus.Connector.Chan()

loop:
	for {
		// wait for shutdown
		log.Info("waiting…")

		select {
		case <-shutdown:
			break loop
		case item := <-queue:
			conn.log.Infof("received control: %s  public key: %x  connect: %x", item.Command, item.Parameters[0], item.Parameters[1])
			connectToUpstream(conn.log, conn.clients, conn.dynamicStart, item.Command, item.Parameters[0], item.Parameters[1])

		case <-time.After(cycleInterval):
			conn.process()
		}
	}
	log.Info("shutting down…")
	for _, client := range conn.clients {
		client.Destroy()
	}
	log.Info("stopped")
}

// process the connect and return response
func (conn *connector) process() {
	// run the machine until it pauses
	for conn.runStateMachine() {
	}
}

// run state machine
// return:
//   true  if want more cycles
//   false to pase for I/O
func (conn *connector) runStateMachine() bool {
	log := conn.log

	log.Infof("current state: %s", conn.state)

	continueLooping := true

	switch conn.state {
	case cStateConnecting:
		mode.Set(mode.Resynchronise)
		clientCount := 0

		for _, client := range conn.clients {
			if client.IsOK() {
				clientCount += 1
			}
		}
		log.Infof("connections: %d", clientCount)
		if clientCount >= minimumClients {
			conn.state += 1
		}
		continueLooping = false

	case cStateHighestBlock:
		conn.height, conn.theClient = getHeight(conn.clients)
		if conn.height > 0 && nil != conn.theClient {
			conn.state += 1
		} else {
			continueLooping = false
		}
		log.Infof("highest block number: %d", conn.height)

	case cStateForkDetect:
		height := block.GetHeight()
		if conn.height <= height {
			conn.state = cStateRebuild
		} else {
			// first block number
			conn.startBlockNumber = genesis.BlockNumber + 1
			conn.state += 1 // assume success
			log.Infof("block number: %d", height)

			// check digests of descending blocks (to detect a fork)
		check_digests:
			for h := height; h > genesis.BlockNumber; h -= 1 {
				digest, err := block.DigestForBlock(h)
				if nil != err {
					log.Infof("block number: %d  local digest error: %v", h, err)
					conn.state = cStateHighestBlock // retry
					break check_digests
				}
				d, err := conn.theClient.GetBlockDigest(h)
				if nil != err {
					log.Infof("block number: %d  fetch digest error: %v", h, err)
					conn.state = cStateHighestBlock // retry
					break check_digests
				} else if d == digest {
					if height-h >= forkProtection {
						conn.state = cStateHighestBlock
						break check_digests
					}
					conn.startBlockNumber = h + 1
					log.Infof("fork from block number: %d", conn.startBlockNumber)

					// remove old blocks
					err := block.DeleteDownToBlock(conn.startBlockNumber)
					if nil != err {
						log.Errorf("delete down to block number: %d  error: %v", conn.startBlockNumber, err)
						conn.state = cStateHighestBlock // retry
					}
					break check_digests
				}
			}
		}

	case cStateFetchBlocks:

		continueLooping = false

	fetch_blocks:
		for n := 0; n < fetchBlocksPerCycle; n += 1 {

			if conn.startBlockNumber > conn.height {
				conn.state = cStateHighestBlock // just in case block height has changed
				continueLooping = true
				break fetch_blocks
			}

			log.Infof("fetch block number: %d", conn.startBlockNumber)
			packedBlock, err := conn.theClient.GetBlockData(conn.startBlockNumber)
			if nil != err {
				log.Errorf("fetch block number: %d  error: %v", conn.startBlockNumber, err)
				conn.state = cStateHighestBlock // retry
				break fetch_blocks
			}
			log.Debugf("store block number: %d", conn.startBlockNumber)
			err = block.StoreIncoming(packedBlock)
			if nil != err {
				log.Errorf("store block number: %d  error: %v", conn.startBlockNumber, err)
				conn.state = cStateHighestBlock // retry
				break fetch_blocks
			}

			// next block
			conn.startBlockNumber += 1

		}

	case cStateRebuild:
		// return to normal operations
		conn.state += 1  // next state
		conn.samples = 0 // zero out the counter
		mode.Set(mode.Normal)
		continueLooping = false

	case cStateSampling:
		// check peers
		conn.height, conn.theClient = getHeight(conn.clients)
		height := block.GetHeight()

		log.Infof("height  remote: %d  local: %d", conn.height, height)

		continueLooping = false

		if conn.height > height {
			if conn.height-height >= 2 {
				conn.state = cStateForkDetect
				continueLooping = true
			} else {
				conn.samples += 1
				if conn.samples > samplelingLimit {
					conn.state = cStateForkDetect
					continueLooping = true
				}
			}
		}
	}
	return continueLooping
}

func getHeight(clients []*upstream.Upstream) (height uint64, theClient *upstream.Upstream) {
	theClient = nil
	height = 0
	for _, client := range clients {
		h := client.GetHeight()
		if h > height {
			height = h
			theClient = client
		}
	}
	return height, theClient
}

func (state connectorState) String() string {
	switch state {
	case cStateConnecting:
		return "Connecting"
	case cStateHighestBlock:
		return "HighestBlock"
	case cStateForkDetect:
		return "ForkDetect"
	case cStateFetchBlocks:
		return "FetchBlocks"
	case cStateRebuild:
		return "Rebuild"
	case cStateSampling:
		return "Sampling"
	default:
		return "*Unknown*"
	}
}
