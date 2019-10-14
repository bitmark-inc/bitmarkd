// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package peer

import (
	"bytes"
	"container/list"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/blockheader"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/genesis"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/peer/upstream"
	"github.com/bitmark-inc/bitmarkd/peer/voting"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/logger"
)

// various timeouts
const (
	// pause to limit bandwidth
	cycleInterval = 15 * time.Second

	// time out for connections
	connectorTimeout = 60 * time.Second

	// number of cycles to be 1 block out of sync before resync
	samplingLimit = 6

	// number of blocks to fetch in one set
	fetchBlocksPerCycle = 200

	// fail to fork if height difference is greater than this
	forkProtection = 60

	// do not proceed unless this many clients are connected
	minimumClients = 5

	// total number of dynamic clients
	maximumDynamicClients = 25

	// client should exist at least 1 response with in this number
	activeTime = 60 * time.Second

	// fast sync option to fetch block
	fastSyncFetchBlocksPerCycle = 2000
	fastSyncSkipPerBlocks       = 100
	fastSyncPivotBlocks         = 1000
)

type connector struct {
	sync.RWMutex

	log        *logger.L
	preferIPv6 bool

	staticClients []upstream.Upstream

	dynamicClients list.List

	state connectorState

	theClient        upstream.Upstream // client used for fetching blocks
	startBlockNumber uint64            // block number where local chain forks
	height           uint64            // block number on best node
	samples          int               // counter to detect missed block broadcast
	votes            voting.Voting

	fastSyncEnabled bool   // fast sync mode enabled?
	blocksPerCycle  int    // number of blocks to fetch per cycle
	pivotPoint      uint64 // block number to stop fast syncing
}

// initialise the connector
func (conn *connector) initialise(
	privateKey []byte,
	publicKey []byte,
	connect []Connection,
	dynamicEnabled bool,
	preferIPv6 bool,
	fastSync bool,
) error {

	log := logger.New("connector")
	conn.log = log

	conn.preferIPv6 = preferIPv6

	conn.fastSyncEnabled = fastSync

	log.Info("initialising…")

	// allocate all sockets
	staticCount := len(connect) // can be zero
	if 0 == staticCount && !dynamicEnabled {
		log.Error("zero static connections and dynamic is disabled")
		return fault.NoConnectionsAvailable
	}
	conn.staticClients = make([]upstream.Upstream, staticCount)

	// initially connect all static sockets
	wg := sync.WaitGroup{}
	errCh := make(chan error, len(connect))

	conn.log.Debugf("static connection count: %d", len(connect))

	for i, c := range connect {
		wg.Add(1)

		// start new goroutine for each connection
		go func(conn *connector, c Connection, i int, wg *sync.WaitGroup, ch chan error) {

			// error function call
			errF := func(wg *sync.WaitGroup, ch chan error, e error) {
				ch <- e
				wg.Done()
			}

			// for canonicaling the error
			canonicalErrF := func(c Connection, e error) error {
				return fmt.Errorf("client: %q error: %s", c.Address, e)
			}

			address, err := util.NewConnection(c.Address)
			if nil != err {
				log.Errorf("client[%d]=address: %q  error: %s", i, c.Address, err)
				errF(wg, ch, canonicalErrF(c, err))
				return
			}
			serverPublicKey, err := zmqutil.ReadPublicKey(c.PublicKey)
			if nil != err {
				log.Errorf("client[%d]=public: %q  error: %s", i, c.PublicKey, err)
				errF(wg, ch, canonicalErrF(c, err))
				return
			}

			// prevent connection to self
			if bytes.Equal(publicKey, serverPublicKey) {
				err := fault.ConnectingToSelfForbidden
				log.Errorf("client[%d]=public: %q  error: %s", i, c.PublicKey, err)
				errF(wg, ch, canonicalErrF(c, err))
				return
			}

			client, err := upstream.New(privateKey, publicKey, connectorTimeout)
			if nil != err {
				log.Errorf("client[%d]=%q  error: %s", i, address, err)
				errF(wg, ch, canonicalErrF(c, err))
				return
			}

			conn.Lock()
			conn.staticClients[i] = client
			globalData.connectorClients = append(globalData.connectorClients, client)
			conn.Unlock()

			err = client.Connect(address, serverPublicKey)
			if nil != err {
				log.Errorf("connect[%d]=%q  error: %s", i, address, err)
				errF(wg, ch, canonicalErrF(c, err))
				return
			}
			log.Infof("public key: %x  at: %q", serverPublicKey, c.Address)
			wg.Done()

		}(conn, c, i, &wg, errCh)
	}

	conn.log.Debug("waiting for all static connections...")
	wg.Wait()

	// drop error channel for getting all errors
	errs := make([]error, 0)
	for len(errCh) > 0 {
		errs = append(errs, <-errCh)
	}

	// error code for goto fail
	err := error(nil)

	if len(errs) == 1 {
		err = errs[0]
		goto fail
	} else if len(errs) > 1 {
		err = compositeError(errs)
		goto fail
	}

	// just create sockets for dynamic clients
	for i := 0; i < maximumDynamicClients; i++ {
		client, e := upstream.New(privateKey, publicKey, connectorTimeout)
		if nil != err {
			log.Errorf("client[%d]  error: %s", i, e)
			err = e
			goto fail
		}

		// create list of all dynamic clients
		conn.dynamicClients.PushBack(client)

		globalData.connectorClients = append(globalData.connectorClients, client)
	}

	conn.votes = voting.NewVoting()

	// start state machine
	conn.nextState(cStateConnecting)

	return nil

	// error handling
fail:
	conn.destroy()

	return err
}

// combine multi error into one
func compositeError(errors []error) error {
	if nil == errors || 0 == len(errors) {
		return nil
	}
	var ce strings.Builder
	ce.WriteString("composite error: [")
	len := len(errors)
	for i, e := range errors {
		ce.WriteString(e.Error())
		if i < len-1 {
			ce.WriteString(", ")
		}
	}
	ce.WriteString("]")
	return fmt.Errorf(ce.String())
}

func (conn *connector) allClients(
	f func(client upstream.Upstream, e *list.Element),
) {
	for _, client := range conn.staticClients {
		f(client, nil)
	}
	for e := conn.dynamicClients.Front(); nil != e; e = e.Next() {
		f(e.Value.(upstream.Upstream), e)
	}
}

func (conn *connector) searchClients(
	f func(client upstream.Upstream, e *list.Element) bool,
) {
	for _, client := range conn.staticClients {
		if f(client, nil) {
			return
		}
	}
	for e := conn.dynamicClients.Front(); nil != e; e = e.Next() {
		if f(e.Value.(upstream.Upstream), e) {
			return
		}
	}
}

func (conn *connector) destroy() {
	conn.allClients(func(client upstream.Upstream, e *list.Element) {
		client.Destroy()
	})
}

// various RPC calls to upstream connections
func (conn *connector) Run(args interface{}, shutdown <-chan struct{}) {
	log := conn.log

	log.Info("starting…")

	queue := messagebus.Bus.Connector.Chan()

	timer := time.After(cycleInterval)

loop:
	for {
		// wait for shutdown
		log.Debug("waiting…")

		select {
		case <-shutdown:
			break loop
		case <-timer: // timer has priority over queue
			timer = time.After(cycleInterval)
			conn.process()
		case item := <-queue:
			c, _ := util.PackedConnection(item.Parameters[1]).Unpack()
			conn.log.Debugf(
				"received control: %s  public key: %x  connect: %x %q",
				item.Command,
				item.Parameters[0],
				item.Parameters[1],
				c,
			)

			switch item.Command {
			case "@D": // internal command: delete a peer
				conn.releaseServerKey(item.Parameters[0])
				conn.log.Infof(
					"connector receive server public key: %x",
					item.Parameters[0],
				)
			default:
				err := conn.connectUpstream(
					item.Command,
					item.Parameters[0],
					item.Parameters[1],
				)
				if nil != err {
					conn.log.Warnf("connect upstream error: %s", err)
				}
			}
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
		globalData.clientCount = conn.getConnectedClientCount()
		log.Infof("connections: %d", globalData.clientCount)

		if isConnectionEnough(globalData.clientCount) {
			conn.nextState(cStateHighestBlock)
		} else {
			log.Warnf("connections: %d below minimum client count: %d", globalData.clientCount, minimumClients)
			messagebus.Bus.Announce.Send("reconnect")
		}
		continueLooping = false

	case cStateHighestBlock:
		if conn.updateHeightAndClient() {
			log.Infof("highest block number: %d  client: %s", conn.height, conn.theClient.Name())
			if conn.hasBetterChain(blockheader.Height()) {
				log.Infof("new chain from %s, height %d, digest %s", conn.theClient.Name(), conn.height, conn.theClient.CachedRemoteDigestOfLocalHeight().String())
				log.Info("enter fork detect state")
				conn.nextState(cStateForkDetect)
			} else if conn.isSameChain() {
				log.Info("remote same chain")
				conn.nextState(cStateRebuild)
			} else {
				log.Info("remote chain invalid, stop looping for now")
				continueLooping = false
			}
		} else {
			log.Warn("highest block: connection lost")
			conn.nextState(cStateConnecting)
			continueLooping = false
		}

	case cStateForkDetect:
		height := blockheader.Height()
		if !conn.hasBetterChain(height) {
			log.Info("remote without better chain, enter state rebuild")
			conn.nextState(cStateRebuild)
		} else {
			// determine pivot point to stop fast sync
			if conn.height > fastSyncPivotBlocks {
				conn.pivotPoint = conn.height - fastSyncPivotBlocks
			} else {
				conn.pivotPoint = 0
			}

			log.Infof("Pivot point for fast sync: %d", conn.pivotPoint)

			// first block number
			conn.startBlockNumber = genesis.BlockNumber + 1
			conn.nextState(cStateFetchBlocks) // assume success
			log.Infof("local block number: %d", height)

			blockheader.ClearCache()
			// check digests of descending blocks (to detect a fork)
		check_digests:
			for h := height; h >= genesis.BlockNumber; h -= 1 {
				digest, err := blockheader.DigestForBlock(h)
				if nil != err {
					log.Infof("block number: %d  local digest error: %s", h, err)
					conn.nextState(cStateHighestBlock) // retry
					break check_digests
				}
				d, err := conn.theClient.RemoteDigestOfHeight(h)
				if nil != err {
					log.Infof("block number: %d  fetch digest error: %s", h, err)
					conn.nextState(cStateHighestBlock) // retry
					break check_digests
				} else if d == digest {
					if height-h >= forkProtection {
						log.Errorf("fork protection at: %d - %d >= %d", height, h, forkProtection)
						conn.nextState(cStateHighestBlock)
						break check_digests
					}

					conn.startBlockNumber = h + 1
					log.Infof("fork from block number: %d", conn.startBlockNumber)

					// remove old blocks
					err := block.DeleteDownToBlock(conn.startBlockNumber)
					if nil != err {
						log.Errorf("delete down to block number: %d  error: %s", conn.startBlockNumber, err)
						conn.nextState(cStateHighestBlock) // retry
					}
					break check_digests
				}
			}
		}

	case cStateFetchBlocks:
		continueLooping = false
		var packedBlock []byte
		var packedNextBlock []byte

		// Check fast sync state on each loop
		if conn.fastSyncEnabled && conn.pivotPoint >= conn.startBlockNumber+fastSyncFetchBlocksPerCycle {
			conn.blocksPerCycle = fastSyncFetchBlocksPerCycle
		} else {
			conn.blocksPerCycle = fetchBlocksPerCycle
		}

	fetch_blocks:
		for i := 0; i < conn.blocksPerCycle; i++ {
			if conn.startBlockNumber > conn.height {
				// just in case block height has changed
				log.Infof("height changed from: %d to: %d", conn.height, conn.startBlockNumber)
				conn.nextState(cStateHighestBlock)
				continueLooping = true
				break fetch_blocks
			}

			log.Infof("fetch block number: %d", conn.startBlockNumber)
			if packedNextBlock == nil {
				p, err := conn.theClient.GetBlockData(conn.startBlockNumber)
				if nil != err {
					log.Errorf("fetch block number: %d  error: %s", conn.startBlockNumber, err)
					conn.nextState(cStateHighestBlock) // retry
					break fetch_blocks
				}
				packedBlock = p
			} else {
				packedBlock = packedNextBlock
			}

			if conn.fastSyncEnabled {
				// test a random block for forgery
				if i > 0 && i%fastSyncSkipPerBlocks == 0 {
					h := conn.startBlockNumber - uint64(rand.Intn(fastSyncSkipPerBlocks))
					log.Debugf("select random block: %d to test for forgery", h)
					digest, err := blockheader.DigestForBlock(h)
					if nil != err {
						log.Infof("block number: %d  local digest error: %s", h, err)
						conn.nextState(cStateHighestBlock) // retry
						break fetch_blocks
					}
					d, err := conn.theClient.RemoteDigestOfHeight(h)
					if nil != err {
						log.Infof("block number: %d  fetch digest error: %s", h, err)
						conn.nextState(cStateHighestBlock) // retry
						break fetch_blocks
					}

					if d != digest {
						log.Warnf("potetial block forgery: %d", h)

						// remove old blocks
						startingPoint := conn.startBlockNumber - uint64(i)
						err := block.DeleteDownToBlock(startingPoint)
						if nil != err {
							log.Errorf("delete down to block number: %d  error: %s", startingPoint, err)
						}

						conn.fastSyncEnabled = false
						conn.nextState(cStateHighestBlock)
						conn.startBlockNumber = startingPoint
						break fetch_blocks
					}
				}

				// get next block:
				//   packedNextBlock will be nil when local height is same as remote
				var err error
				packedNextBlock, err = conn.theClient.GetBlockData(conn.startBlockNumber + 1)
				if nil != err {
					log.Debugf("fetch next block number: %d  error: %s", conn.startBlockNumber+1, err)
				}
			} else {
				packedNextBlock = nil
			}

			log.Debugf("store block number: %d", conn.startBlockNumber)
			err := block.StoreIncoming(packedBlock, packedNextBlock, block.NoRescanVerified)
			if nil != err {
				log.Errorf(
					"store block number: %d  error: %s",
					conn.startBlockNumber,
					err,
				)
				conn.nextState(cStateHighestBlock) // retry
				break fetch_blocks
			}

			// next block
			conn.startBlockNumber++
		}

	case cStateRebuild:
		// return to normal operations
		conn.nextState(cStateSampling)
		conn.samples = 0 // zero out the counter
		mode.Set(mode.Normal)
		continueLooping = false

	case cStateSampling:
		// check peers
		globalData.clientCount = conn.getConnectedClientCount()
		if !isConnectionEnough(globalData.clientCount) {
			log.Warnf("connections: %d below minimum client count: %d", globalData.clientCount, minimumClients)
			continueLooping = true
			conn.nextState(cStateConnecting)
			return continueLooping
		}

		log.Infof("connections: %d", globalData.clientCount)

		continueLooping = false

		// check height
		if conn.updateHeightAndClient() {
			height := blockheader.Height()

			log.Infof("height remote: %d, local: %d", conn.height, height)

			if conn.hasBetterChain(height) {
				log.Warn("check height: better chain")
				conn.nextState(cStateForkDetect)
				continueLooping = true
			} else {
				conn.samples = 0
			}
		} else {
			conn.samples++
			if conn.samples > samplingLimit {
				log.Warn("check height: time to resync")
				conn.nextState(cStateForkDetect)
				continueLooping = true
			}
		}

	}
	return continueLooping
}

func isConnectionEnough(count int) bool {
	return minimumClients <= count
}

func (conn *connector) isSameChain() bool {
	if conn.theClient == nil {
		conn.log.Debug("remote client empty")
		return false
	}

	localDigest, err := blockheader.DigestForBlock(blockheader.Height())
	if nil != err {
		return false
	}

	if conn.height == blockheader.Height() && conn.theClient.CachedRemoteDigestOfLocalHeight() == localDigest {
		return true
	}

	return false
}

func (conn *connector) hasBetterChain(localHeight uint64) bool {
	if conn.theClient == nil {
		conn.log.Debug("remote client empty")
		return false
	}

	if conn.height < localHeight {
		conn.log.Debugf("remote height %d is shorter than local height %d", conn.height, localHeight)
		return false
	}

	if conn.height == localHeight && !conn.hasSmallerDigestThanLocal(localHeight) {
		return false
	}

	return true
}

// different chain but with same height, possible fork exist
// choose the chain that has smaller digest
func (conn *connector) hasSmallerDigestThanLocal(localHeight uint64) bool {
	remoteDigest := conn.theClient.CachedRemoteDigestOfLocalHeight()

	// if upstream update during processing
	if conn.theClient.LocalHeight() != localHeight {
		conn.log.Warnf("remote height %d is different than local height %d", conn.theClient.LocalHeight(), localHeight)
		return false
	}

	localDigest, err := blockheader.DigestForBlock(localHeight)
	if nil != err {
		conn.log.Warnf("local height: %d  digest error: %s", localHeight, err)
		return false
	}

	return remoteDigest.SmallerDigestThan(localDigest)
}

func (conn *connector) updateHeightAndClient() bool {
	conn.votes.Reset()
	conn.votes.SetMinHeight(blockheader.Height())
	conn.startElection()
	elected, height := conn.elected()
	if 0 == height {
		conn.height = 0
		return false
	}

	winnerName := elected.Name()
	remoteAddr, err := elected.RemoteAddr()
	if nil != err {
		conn.log.Warnf("%s socket not connected", winnerName)
		conn.height = 0
		return false
	}

	conn.log.Debugf("winner %s majority height %d, connect to %s",
		winnerName,
		height,
		remoteAddr,
	)

	if height > 0 && nil != elected {
		globalData.blockHeight = height
	}
	conn.theClient = elected
	conn.height = height
	return true
}

func (conn *connector) startElection() {
	conn.allClients(func(client upstream.Upstream, e *list.Element) {
		if client.IsConnected() && client.ActiveInThePast(activeTime) {
			conn.votes.VoteBy(client)
		}
	})
}

func (conn *connector) elected() (upstream.Upstream, uint64) {
	elected, height, err := conn.votes.ElectedCandidate()
	if nil != err {
		conn.log.Errorf("get elected with error: %s", err)
		return nil, 0
	}

	remoteAddr, err := elected.RemoteAddr()
	if nil != err {
		conn.log.Errorf("get client string with error: %s", err)
		return nil, 0
	}

	digest := elected.CachedRemoteDigestOfLocalHeight()
	conn.log.Infof(
		"digest: %s elected with %d votes, remote addr: %s, height: %d",
		digest,
		conn.votes.NumVoteOfDigest(digest),
		remoteAddr,
		height,
	)

	return elected, height
}

func (conn *connector) connectUpstream(
	priority string,
	serverPublicKey []byte,
	addresses []byte,
) error {

	log := conn.log

	log.Debugf("connect: %s to: %x @ %x", priority, serverPublicKey, addresses)

	// extract the first valid address
	connV4, connV6 := util.PackedConnection(addresses).Unpack46()

	// need to know if this node has IPv6
	address := connV4
	if nil != connV6 && conn.preferIPv6 {
		address = connV6
	}

	if nil == address {
		log.Errorf(
			"reconnect: %x  error: no suitable address found ipv6 allowed: %t",
			serverPublicKey,
			conn.preferIPv6,
		)
		return fault.AddressIsNil
	}

	log.Infof("connect: %s to: %x @ %s", priority, serverPublicKey, address)

	// see if already connected to this node
	alreadyConnected := false
	conn.searchClients(func(client upstream.Upstream, e *list.Element) bool {
		if client.IsConnectedTo(serverPublicKey) {
			if nil == e {
				log.Debugf(
					"already have static connection to: %x @ %s",
					serverPublicKey,
					*address,
				)
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
	client := conn.dynamicClients.Front().Value.(upstream.Upstream)
	err := client.Connect(address, serverPublicKey)
	if nil != err {
		log.Errorf("ConnectTo: %x @ %s  error: %s", serverPublicKey, *address, err)
	} else {
		conn.dynamicClients.MoveToBack(conn.dynamicClients.Front())
	}

	return err
}

func (conn *connector) releaseServerKey(serverPublicKey []byte) error {
	log := conn.log
	conn.searchClients(func(client upstream.Upstream, e *list.Element) bool {
		if bytes.Equal(serverPublicKey, client.ServerPublicKey()) {
			if e == nil { // static Clients
				log.Infof("refuse to delete static peer: %x", serverPublicKey)
			} else { // dynamic Clients
				client.ResetServer()
				log.Infof("peer: %x is released in upstream", serverPublicKey)
				return true
			}
		}
		return false
	})
	return nil
}

func (conn *connector) nextState(newState connectorState) {
	conn.state = newState
}

func (conn *connector) getConnectedClientCount() int {
	clientCount := 0
	conn.allClients(func(client upstream.Upstream, e *list.Element) {
		if client.IsConnected() {
			clientCount++
		}
	})
	return clientCount
}
