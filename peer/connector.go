// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package peer

import (
	"bytes"
	"container/list"
	"encoding/hex"
	"fmt"
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
	"github.com/bitmark-inc/logger"
)

// various timeouts
const (
	// pause to limit bandwidth
	cycleInterval = 15 * time.Second

	// time out for connections
	connectorTimeout = 60 * time.Second

	// number of cycles to be 1 block out of sync before resync
	samplelingLimit = 10

	// number of blocks to fetch in one set
	fetchBlocksPerCycle = 200

	// fail to fork if height difference is greater than this
	forkProtection = 60

	// do not proceed unless this many clients are connected
	minimumClients = 5

	// total number of dynamic clients
	maximumDynamicClients = 10

	// client should exist at least 1 response with in this number
	activePastSec = 60

	// fast sync option to fetch block
	fastSyncFetchBlocksPerCycle = 2048
	fastSyncSkipPerBlocks       = 100
)

type ConnectorIntf interface {
	PrintUpstreams(string) string
	Run(interface{}, <-chan struct{})
}

type connector struct {
	ConnectorIntf

	sync.RWMutex
	log        *logger.L
	preferIPv6 bool

	staticClients []upstream.UpstreamIntf

	dynamicClients list.List

	state connectorState

	theClient        upstream.UpstreamIntf // client used for fetching blocks
	startBlockNumber uint64                // block number where local chain forks
	height           uint64                // block number on best node
	samples          int                   // counter to detect missed block broadcast
	votes            voting.Voting

	blockPerCycle   int    // number of blocks to fetch per cycle
	blockCycleIndex int    // current index of block is fetching in cycle
	pivotPoint      uint64 // block number to stop fast syncing
}

// initialise the connector
func (conn *connector) initialise(
	privateKey []byte,
	publicKey []byte,
	connect []Connection,
	dynamicEnabled bool,
	preferIPv6 bool,
) error {

	log := logger.New("connector")
	conn.log = log

	conn.preferIPv6 = preferIPv6

	log.Info("initialising…")

	// allocate all sockets
	staticCount := len(connect) // can be zero
	if 0 == staticCount && !dynamicEnabled {
		log.Error("zero static connections and dynamic is disabled")
		return fault.ErrNoConnectionsAvailable
	}
	conn.staticClients = make([]upstream.UpstreamIntf, staticCount)

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
			serverPublicKey, err := hex.DecodeString(c.PublicKey)
			if nil != err {
				log.Errorf("client[%d]=public: %q  error: %s", i, c.PublicKey, err)
				errF(wg, ch, canonicalErrF(c, err))
				return
			}

			// prevent connection to self
			if bytes.Equal(publicKey, serverPublicKey) {
				err := fault.ErrConnectingToSelfForbidden
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
	conn.toState(cStateConnecting)

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
	f func(client upstream.UpstreamIntf, e *list.Element),
) {
	for _, client := range conn.staticClients {
		f(client, nil)
	}
	for e := conn.dynamicClients.Front(); nil != e; e = e.Next() {
		f(e.Value.(upstream.UpstreamIntf), e)
	}
}

func (conn *connector) searchClients(
	f func(client upstream.UpstreamIntf, e *list.Element) bool,
) {
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
	conn.allClients(func(client upstream.UpstreamIntf, e *list.Element) {
		client.Destroy()
	})
}

// Print all upstream connectors default: "debug",
// available: "debug", "info" , "warn" , used for debug
func (conn *connector) PrintUpstreams(prefix string) string {
	counter := 0
	upstreams := ""
	conn.allClients(func(client upstream.UpstreamIntf, e *list.Element) {
		counter++
		upstreams = fmt.Sprintf("%s%supstream%d: %v\n", upstreams, prefix, counter, client)
	})
	return upstreams
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
			conn.nextState()
		} else {
			log.Warnf("connections: %d below minimum client count: %d", globalData.clientCount, minimumClients)
			messagebus.Bus.Announce.Send("reconnect")
		}
		continueLooping = false

	case cStateHighestBlock:
		conn.height, conn.theClient = conn.getHeightAndClient()
		if conn.hasBetterChain(blockheader.Height()) {
			log.Infof("new chain from %s, height %d, digest %s", conn.theClient.Name(), conn.height, conn.theClient.CachedRemoteDigestOfLocalHeight().String())
			log.Info("enter fork detect state")
			conn.nextState()
		} else if conn.isSameChain() {
			log.Info("remote same chain")
			conn.toState(cStateRebuild)
		} else {
			log.Info("remote chain invalid, stop looping for now")
			continueLooping = false
		}
		log.Infof("highest block number: %d", conn.height)

	case cStateForkDetect:
		height := blockheader.Height()
		if !conn.hasBetterChain(height) {
			log.Debug("remote without better chain, enter state rebuild")
			conn.toState(cStateRebuild)
		} else {
			// determine pivot point to stop fast sync
			conn.pivotPoint = conn.height - 1024
			log.Debugf("Pivot point for fast sync: %d", conn.pivotPoint)

			// first block number
			conn.startBlockNumber = genesis.BlockNumber + 1
			conn.nextState() // assume success
			log.Infof("local block number: %d", height)

			// check digests of descending blocks (to detect a fork)
		check_digests:
			for h := height; h >= genesis.BlockNumber; h -= 1 {
				digest, err := blockheader.DigestForBlock(h)
				if nil != err {
					log.Infof("block number: %d  local digest error: %s", h, err)
					conn.toState(cStateHighestBlock) // retry
					break check_digests
				}
				d, err := conn.theClient.RemoteDigestOfHeight(h)
				if nil != err {
					log.Infof("block number: %d  fetch digest error: %s", h, err)
					conn.toState(cStateHighestBlock) // retry
					break check_digests
				} else if d == digest {
					if height-h >= forkProtection {
						log.Errorf("fork protection at: %d - %d >= %d", height, h, forkProtection)
						conn.toState(cStateHighestBlock)
						break check_digests
					}

					conn.startBlockNumber = h + 1
					log.Infof("fork from block number: %d", conn.startBlockNumber)

					// remove old blocks
					err := block.DeleteDownToBlock(conn.startBlockNumber)
					if nil != err {
						log.Errorf("delete down to block number: %d  error: %s", conn.startBlockNumber, err)
						conn.toState(cStateHighestBlock) // retry
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
		conn.enableFastSyncIfNeeded()

	fetch_blocks:
		for conn.blockCycleIndex = 0; conn.blockCycleIndex < conn.blockPerCycle; conn.blockCycleIndex++ {
			log.Debugf("block cycle index: %d", conn.blockCycleIndex)
			if conn.startBlockNumber > conn.height {
				// just in case block height has changed
				conn.toState(cStateHighestBlock)
				continueLooping = true
				break fetch_blocks
			}

			log.Infof("fetch block number: %d", conn.startBlockNumber)
			if packedNextBlock == nil {
				p, err := conn.theClient.GetBlockData(conn.startBlockNumber)
				if nil != err {
					log.Errorf("fetch block number: %d  error: %s", conn.startBlockNumber, err)
					conn.toState(cStateHighestBlock) // retry
					break fetch_blocks
				}
				packedBlock = p
			} else {
				packedBlock = packedNextBlock
			}

			// get next block
			if conn.blockCycleIndex%fastSyncSkipPerBlocks != 0 && mode.Is(mode.Fastsynchronise) {
				nextBlock, err := conn.theClient.GetBlockData(conn.startBlockNumber + 1)
				if nil != err {
					log.Errorf("fetch block number: %d  error: %s", conn.startBlockNumber+1, err)
					conn.toState(cStateHighestBlock) // retry
					break fetch_blocks
				}
				packedNextBlock = nextBlock
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
				conn.toState(cStateHighestBlock) // retry
				break fetch_blocks
			}

			// next block
			conn.startBlockNumber++

		}

	case cStateRebuild:
		// return to normal operations
		conn.nextState()
		conn.samples = 0 // zero out the counter
		mode.Set(mode.Normal)
		continueLooping = false

	case cStateSampling:
		// check peers
		globalData.clientCount = conn.getConnectedClientCount()
		if !isConnectionEnough(globalData.clientCount) {
			log.Warnf("connections: %d below minimum client count: %d", globalData.clientCount, minimumClients)
			continueLooping = true
			conn.toState(cStateConnecting)
			return continueLooping
		}

		log.Infof("connections: %d", globalData.clientCount)

		// check height
		conn.height, conn.theClient = conn.getHeightAndClient()
		height := blockheader.Height()

		log.Infof("height remote: %d, local: %d", conn.height, height)

		continueLooping = false

		if conn.hasBetterChain(height) {
			conn.toState(cStateForkDetect)
			continueLooping = true
		} else {
			conn.samples++
			if conn.samples > samplelingLimit {
				conn.toState(cStateForkDetect)
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

	if conn.height == localHeight && !conn.hasSamllerDigestThanLocal(localHeight) {
		return false
	}

	return true
}

// different chain but with same height, possible fork exist
// choose the chain that has smaller digest
func (conn *connector) hasSamllerDigestThanLocal(localHeight uint64) bool {
	remoteDigest := conn.theClient.CachedRemoteDigestOfLocalHeight()
	// if upstream update during processing
	if conn.theClient.LocalHeight() != localHeight {
		conn.log.Warnf("remote height %d is different than local height %d", conn.theClient.LocalHeight(), localHeight)
		return false
	}

	localDigest, err := blockheader.DigestForBlock(localHeight)
	if nil != err {
		return false
	}

	return remoteDigest.SmallerDigestThan(localDigest)
}

func (conn *connector) getHeightAndClient() (uint64, upstream.UpstreamIntf) {
	conn.votes.Reset()
	conn.votes.SetMinHeight(blockheader.Height())
	conn.startElection()
	elected, height := conn.elected()
	if uint64(0) == height {
		return uint64(0), nil
	}

	winnerName := elected.Name()
	remoteAddr, err := elected.RemoteAddr()
	if nil != err {
		conn.log.Warnf("%s socket not connected", winnerName)
		return uint64(0), nil
	}

	conn.log.Debugf("winner %s majority height %d, connect to %s",
		winnerName,
		height,
		remoteAddr,
	)

	if height > uint64(0) && nil != elected {
		globalData.blockHeight = height
	}
	return height, elected
}

func (conn *connector) startElection() {
	conn.allClients(func(client upstream.UpstreamIntf, e *list.Element) {
		if client.IsConnected() && client.ActiveInPastSeconds(activePastSec) {
			conn.votes.VoteBy(client)
		}
	})
}

func (conn *connector) elected() (upstream.UpstreamIntf, uint64) {
	elected, height, err := conn.votes.ElectedCandidate()
	if nil != err {
		conn.log.Errorf("get elected with error: %s", err.Error())
		return nil, uint64(0)
	}

	remoteAddr, err := elected.RemoteAddr()
	if nil != err {
		conn.log.Errorf("get client string with error: %s", err.Error())
		return nil, uint64(0)
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
		return fault.ErrAddressIsNil
	}

	log.Infof("connect: %s to: %x @ %s", priority, serverPublicKey, address)

	// see if already connected to this node
	alreadyConnected := false
	conn.searchClients(func(client upstream.UpstreamIntf, e *list.Element) bool {
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
	client := conn.dynamicClients.Front().Value.(*upstream.Upstream)
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
	conn.searchClients(func(client upstream.UpstreamIntf, e *list.Element) bool {
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

func (conn *connector) nextState() {
	conn.state++
}

func (conn *connector) toState(newState connectorState) {
	conn.state = newState
}

func (conn *connector) getConnectedClientCount() int {
	clientCount := 0
	conn.allClients(func(client upstream.UpstreamIntf, e *list.Element) {
		if client.IsConnected() {
			clientCount++
		}
	})
	return clientCount
}

func (conn *connector) enableFastSyncIfNeeded() {
	// Security check, stop if pivot point isn't set
	if conn.pivotPoint == 0 {
		return
	}

	// Determine if it's still good for fast sync
	if conn.pivotPoint < conn.startBlockNumber+fastSyncFetchBlocksPerCycle {
		// Stop fast sync from this point
		// since we don't have enough blocks to have full fast sync cycle
		conn.log.Info("Set sync mode to normal")
		mode.Set(mode.Normal)
		conn.blockCycleIndex = fetchBlocksPerCycle
	} else {
		conn.log.Info("Set sync mode to fast")
		mode.Set(mode.Fastsynchronise)
		conn.blockPerCycle = fastSyncFetchBlocksPerCycle
	}
}
