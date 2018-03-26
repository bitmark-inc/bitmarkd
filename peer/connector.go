// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package peer

import (
	"bytes"
	"container/list"
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
	cycleInterval         = 15 * time.Second // pause to limit bandwidth
	connectorTimeout      = 60 * time.Second // time out for connections
	samplelingLimit       = 10               // number of cycles to be 1 block out of sync before resync
	fetchBlocksPerCycle   = 100              // number of blocks to fetch in one set
	forkProtection        = 60               // fail to fork if height difference is greater than this
	minimumClients        = 3                // do not proceed unless this many clients are connected
	maximumDynamicClients = 10               // total number of dynamic clients
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
	log *logger.L

	staticClients []*upstream.Upstream

	dynamicClients list.List

	state connectorState

	theClient        *upstream.Upstream // client used for fetching blocks
	startBlockNumber uint64             // block number where local chain forks
	height           uint64             // block number on best node
	samples          int                // counter to detect missed block broadcast
}

// initialise the connector
func (conn *connector) initialise(privateKey []byte, publicKey []byte, connect []Connection, dynamicEnabled bool) error {

	log := logger.New("connector")
	conn.log = log

	log.Info("initialising…")

	// allocate all sockets
	staticCount := len(connect) // can be zero
	if 0 == staticCount && !dynamicEnabled {
		log.Error("zero static connections and dynamic is disabled")
		return fault.ErrNoConnectionsAvailable
	}
	conn.staticClients = make([]*upstream.Upstream, staticCount)

	// error code for goto fail
	errX := error(nil)

	// initially connect all static sockets
	for i, c := range connect {
		address, err := util.NewConnection(c.Address)
		if nil != err {
			log.Errorf("client[%d]=address: %q  error: %s", i, c.Address, err)
			errX = err
			goto fail
		}
		serverPublicKey, err := hex.DecodeString(c.PublicKey)
		if nil != err {
			log.Errorf("client[%d]=public: %q  error: %s", i, c.PublicKey, err)
			errX = err
			goto fail
		}

		// prevent connection to self
		if bytes.Equal(publicKey, serverPublicKey) {
			errX = fault.ErrConnectingToSelfForbidden
			log.Errorf("client[%d]=public: %q  error: %s", i, c.PublicKey, errX)
			goto fail
		}

		client, err := upstream.New(privateKey, publicKey, connectorTimeout)
		if nil != err {
			log.Errorf("client[%d]=%q  error: %s", i, address, err)
			errX = err
			goto fail
		}

		conn.staticClients[i] = client
		globalData.connectorClients = append(globalData.connectorClients, client)

		err = client.Connect(address, serverPublicKey)
		if nil != err {
			log.Errorf("connect[%d]=%q  error: %s", i, address, err)
			errX = err
			goto fail
		}
		log.Infof("public key: %x  at: %q", serverPublicKey, c.Address)
	}

	// just create sockets for dynamic clients
	for i := 0; i < maximumDynamicClients; i += 1 {
		client, err := upstream.New(privateKey, publicKey, connectorTimeout)
		if nil != err {
			log.Errorf("client[%d]  error: %s", i, err)
			errX = err
			goto fail
		}

		// create list of all dynamic clients
		conn.dynamicClients.PushBack(client)

		globalData.connectorClients = append(globalData.connectorClients, client)
	}

	// start state machine
	conn.state = cStateConnecting

	return nil

	// error handling
fail:
	conn.destroy()

	return errX
}

func (conn *connector) allClients(f func(client *upstream.Upstream, e *list.Element)) {
	for _, client := range conn.staticClients {
		f(client, nil)
	}
	for e := conn.dynamicClients.Front(); nil != e; e = e.Next() {
		f(e.Value.(*upstream.Upstream), e)
	}
}

func (conn *connector) searchClients(f func(client *upstream.Upstream, e *list.Element) bool) {
	for _, client := range conn.staticClients {
		if f(client, nil) {
			return
		}
	}
	for e := conn.dynamicClients.Front(); nil != e; e = e.Next() {
		if f(e.Value.(*upstream.Upstream), e) {
			return
		}
	}
}

func (conn *connector) destroy() {
	conn.allClients(func(client *upstream.Upstream, e *list.Element) {
		client.Destroy()
	})
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
			//connectToUpstream(conn.log, conn.clients, conn.dynamicStart, item.Command, item.Parameters[0], item.Parameters[1])
			conn.connectUpstream(item.Command, item.Parameters[0], item.Parameters[1])

		case <-time.After(cycleInterval):
			conn.process()
		}
	}
	log.Info("shutting down…")
	conn.destroy()
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

		conn.allClients(func(client *upstream.Upstream, e *list.Element) {
			if client.IsOK() {

				clientCount += 1
			}
		})

		log.Infof("connections: %d", clientCount)
		globalData.clientCount = clientCount
		if clientCount >= minimumClients {
			conn.state += 1
		} else {
			log.Warnf("can not reach the minimum client counts")
			messagebus.Bus.Announce.Send("reconnect")
		}
		continueLooping = false

	case cStateHighestBlock:
		conn.height, conn.theClient = getHeight(conn)
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
					log.Infof("block number: %d  local digest error: %s", h, err)
					conn.state = cStateHighestBlock // retry
					break check_digests
				}
				d, err := conn.theClient.GetBlockDigest(h)
				if nil != err {
					log.Infof("block number: %d  fetch digest error: %s", h, err)
					conn.state = cStateHighestBlock // retry
					break check_digests
				} else if d == digest {
					if height-h >= forkProtection {
						log.Errorf("fork protection at: %d - %d >= %d", height, h, forkProtection)
						conn.state = cStateHighestBlock
						break check_digests
					}
					conn.startBlockNumber = h + 1
					log.Infof("fork from block number: %d", conn.startBlockNumber)

					// remove old blocks
					err := block.DeleteDownToBlock(conn.startBlockNumber)
					if nil != err {
						log.Errorf("delete down to block number: %d  error: %s", conn.startBlockNumber, err)
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
				log.Errorf("fetch block number: %d  error: %s", conn.startBlockNumber, err)
				conn.state = cStateHighestBlock // retry
				break fetch_blocks
			}
			log.Debugf("store block number: %d", conn.startBlockNumber)
			err = block.StoreIncoming(packedBlock)
			if nil != err {
				log.Errorf("store block number: %d  error: %s", conn.startBlockNumber, err)
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
		conn.height, conn.theClient = getHeight(conn)
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

func getHeight(conn *connector) (height uint64, theClient *upstream.Upstream) {
	theClient = nil
	height = 0

	conn.allClients(func(client *upstream.Upstream, e *list.Element) {
		h := client.GetHeight()
		if h > height {
			height = h
			theClient = client
		}
	})

	globalData.blockHeight = height
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

func (conn *connector) connectUpstream(priority string, serverPublicKey []byte, addresses []byte) error {

	log := conn.log

	log.Infof("connect: %s to: %x @ %x", priority, serverPublicKey, addresses)

	// extract the first valid address
	var address *util.Connection

extract_addresses:
	for {
		conn, n := util.PackedConnection(addresses).Unpack()
		addresses = addresses[n:]

		// ***** FIX THIS: could select for IPv4 or IPv6 here
		// ***** FIX THIS: need to get preference e.g. if have IPv6 the prefer IPv6
		if nil != conn {
			address = conn
			break extract_addresses
		}
		if n <= 0 {
			break extract_addresses
		}
		log.Errorf("reconnect: %x (conn: %x)  error: address is nil", serverPublicKey, conn)
	}

	if nil == address {
		log.Errorf("reconnect: %s  error: no addresses found", serverPublicKey)
		return fault.ErrAddressIsNil
	}

	// see if already connected to this node
	alreadyConnected := false
	conn.searchClients(func(client *upstream.Upstream, e *list.Element) bool {
		if client.IsConnectedTo(serverPublicKey) {
			if nil == e {
				log.Debugf("already have static connection to: %x @ %s", serverPublicKey, *address)
			} else {
				log.Debugf("ignore change to: %x @ %s", serverPublicKey, *address)
				conn.dynamicClients.MoveToBack(e)
			}
			alreadyConnected = true
			return true
		}
		return false
	})

	if alreadyConnected {
		return nil
	}

	// reconnect the oldest entry to new node
	log.Infof("reconnect: %x @ %s", serverPublicKey, *address)
	client := conn.dynamicClients.Front().Value.(*upstream.Upstream)
	err := client.Connect(address, serverPublicKey)
	if nil != err {
		log.Errorf("ConnectTo: %x @ %s  error: %s", serverPublicKey, *address, err)
	} else {
		conn.dynamicClients.MoveToBack(conn.dynamicClients.Front())
	}

	return err
}
