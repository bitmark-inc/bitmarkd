// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package consensus

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/blockheader"
	"github.com/bitmark-inc/bitmarkd/consensus/voting"
	"github.com/bitmark-inc/bitmarkd/counter"
	"github.com/bitmark-inc/bitmarkd/genesis"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/p2p"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/logger"
)

// various timeouts
const (
	// pause to limit bandwidth
	cycleInterval = 15 * time.Second

	// number of cycles to be 1 block out of sync before into resync mode
	samplingLimit = 6

	// number of blocks to fetch in one set
	fetchBlocksPerCycle = 200

	// fail to fork if height difference is greater than this
	forkProtection = 60

	// do not proceed unless this many clients are connected
	minimumClients = 5

	// client should exist at least 1 response with in this number
	activeInterval = 60

	// fast sync option to fetch block
	fastSyncFetchBlocksPerCycle = 2000
	fastSyncSkipPerBlocks       = 100
	fastSyncPivotBlocks         = 1000
)

// Machine voting consensus state machine
type Machine struct {
	log              *logger.L
	attachedNode     *p2p.Node
	votingMetrics    *MetricsPeersVoting
	votes            voting.Voting
	targetNode       voting.Candidate //voting winner
	targetHeight     uint64           //voting winner block height
	startBlockNumber uint64
	samples          int
	fastsyncEnabled  bool   // fast sync mode enabled?
	blocksPerCycle   int    // number of blocks to fetch per cycle
	pivotPoint       uint64 // block number to stop fast syncing
	state
}

// NewConsensusMachine get a new StateMachine
func NewConsensusMachine(node *p2p.Node, metric *MetricsPeersVoting, fastsync bool) *Machine {
	machine := &Machine{log: globalData.Log, votingMetrics: metric, attachedNode: node}
	machine.nextState(cStateConnecting)
	machine.votes = voting.NewVoting()
	machine.fastsyncEnabled = fastsync
	return machine
}

//Run Run A ConsensusMachine
func (m *Machine) Run(_ interface{}, shutdown <-chan struct{}) {
	log := m.log
	log.Info("starting a consensus state machine…")
	timer := time.After(machineRunInitial)
loop:
	for {
		// wait for shutdown
		log.Debug("waiting…")
		select {
		case <-shutdown:
			break loop
		case <-timer: // timer has priority over queue
			timer = time.After(cycleInterval)
			m.start()
		}
	}
	log.Info("shutting down…")
	log.Info("stopped")
}

func (m *Machine) start() {
	for !m.transitions() {
	}
}

func (m *Machine) transitions() bool {
	log := m.log
	log.Debugf("current state: %s", m.state)
	stop := false
	switch m.state {
	case cStateConnecting:
		mode.Set(mode.Resynchronise)
		util.LogInfo(log, util.CoYellow, fmt.Sprintf("Enter Connecting state, mode:%s", mode.String()))
		if isConnectionEnough(m.attachedNode.MetricsNetwork.GetConnCount()) {
			m.nextState(cStateHighestBlock)
		} else {
			log.Debugf("connections: %d below minimum client count: %d", uint64(m.attachedNode.MetricsNetwork.GetConnCount()), minimumClients)
		}
		stop = true

	case cStateHighestBlock:
		if m.updateHeightAndClient() {
			log.Infof("highest block number: %d  client: %s", m.targetHeight, m.targetNode.Name())
			if m.hasBetterChain(blockheader.Height()) {
				log.Infof("new chain from %s, height %d, digest %s", m.targetNode.Name(), m.targetHeight, m.targetNode.CachedRemoteDigestOfLocalHeight().String())
				log.Info("enter fork detect state")
				m.nextState(cStateForkDetect)
			} else if m.isSameChain() {
				log.Info("remote same chain")
				m.nextState(cStateRebuild)
			} else {
				log.Info("remote chain invalid, stop looping for now")
				stop = true
			}
		} else {
			log.Warn("highest block: connection lost")
			m.nextState(cStateConnecting)
			stop = true
		}

	case cStateForkDetect:
		util.LogInfo(log, util.CoYellow, fmt.Sprintf("Enter ForkDetect state, mode:%s", mode.String()))
		height := blockheader.Height()
		if !m.hasBetterChain(height) {
			log.Info("remote without better chain, enter state rebuild")
			m.nextState(cStateRebuild)
		} else {
			// determine pivot point to stop fast sync
			if m.targetHeight > fastSyncPivotBlocks {
				m.pivotPoint = m.targetHeight - fastSyncPivotBlocks
			} else {
				m.pivotPoint = 0
			}

			log.Infof("Pivot point for fast sync: %d", m.pivotPoint)

			// first block number
			m.startBlockNumber = genesis.BlockNumber + 1
			m.nextState(cStateFetchBlocks) // assume success
			log.Infof("local block number: %d", height)

			blockheader.ClearCache()
			// check digests of descending blocks (to detect a fork)

		checkDigests:
			for h := height; h >= genesis.BlockNumber; h -= 1 {
				digest, err := blockheader.DigestForBlock(h)
				if nil != err {
					log.Infof("block number: %d  local digest error: %s", h, err)
					m.nextState(cStateHighestBlock) // retry
					break checkDigests
				}
				d, err := m.attachedNode.RemoteDigestOfHeight(m.targetNode.(*P2PCandidatesImpl).ID, h, nil, nil)
				if nil != err {
					log.Infof("block number: %d  fetch digest error: %s", h, err)
					m.nextState(cStateHighestBlock) // retry
					break checkDigests
				} else if d == digest {
					if height-h >= forkProtection {
						log.Errorf("fork protection at: %d - %d >= %d", height, h, forkProtection)
						m.nextState(cStateHighestBlock)
						break checkDigests
					}

					m.startBlockNumber = h + 1
					log.Infof("fork from block number: %d", m.startBlockNumber)

					// remove old blocks
					err := block.DeleteDownToBlock(m.startBlockNumber)
					if nil != err {
						log.Errorf("delete down to block number: %d  error: %s", m.startBlockNumber, err)
						m.nextState(cStateHighestBlock) // retry
					}
					break checkDigests
				}
			}
		}

	case cStateFetchBlocks:
		util.LogInfo(log, util.CoYellow, fmt.Sprintf("Enter FetchBlocks state, mode:%s", mode.String()))
		stop = true
		var packedBlock []byte
		var packedNextBlock []byte

		// Check fast sync state on each loop
		if m.fastsyncEnabled &&
			m.pivotPoint >= m.startBlockNumber+fastSyncFetchBlocksPerCycle {
			m.blocksPerCycle = fastSyncFetchBlocksPerCycle
		} else {
			m.blocksPerCycle = fetchBlocksPerCycle
		}

	fetchBlocks:
		for n := 0; n < m.blocksPerCycle; n++ {
			if m.startBlockNumber > m.targetHeight {
				// just in case block height has changed
				log.Infof("height changed from: %d to: %d", m.targetHeight, m.startBlockNumber)
				m.nextState(cStateHighestBlock)
				stop = false
				break fetchBlocks
			}

			if m.startBlockNumber%100 == 0 {
				log.Warnf("fetch block number: %d", m.startBlockNumber)
			} else {
				log.Infof("fetch block number: %d", m.startBlockNumber)
			}

			if packedNextBlock == nil {
				p, err := m.attachedNode.GetBlockData(m.targetNode.(*P2PCandidatesImpl).ID, m.startBlockNumber, nil, nil)
				if nil != err {
					log.Errorf("fetch block number: %d  error: %s", m.startBlockNumber, err)
					m.nextState(cStateHighestBlock) // retry
					break fetchBlocks
				}
				packedBlock = p
			} else {
				packedBlock = packedNextBlock
			}

			if m.fastsyncEnabled {
				// test a random block for forgery
				if n > 0 && n%fastSyncSkipPerBlocks == 0 {
					h := m.startBlockNumber - uint64(rand.Intn(fastSyncSkipPerBlocks))
					log.Debugf("select random block: %d to test for forgery", h)
					digest, err := blockheader.DigestForBlock(h)
					if nil != err {
						log.Infof("block number: %d  local digest error: %s", h, err)
						m.nextState(cStateHighestBlock) // retry
						break fetchBlocks
					}
					d, err := m.attachedNode.RemoteDigestOfHeight(m.targetNode.(*P2PCandidatesImpl).ID, h, nil, nil)
					if nil != err {
						log.Infof("block number: %d  fetch digest error: %s", h, err)
						m.nextState(cStateHighestBlock) // retry
						break fetchBlocks
					}

					if d != digest {
						log.Warnf("potential block forgery: %d", h)

						// remove old blocks
						startingPoint := m.startBlockNumber - uint64(n)
						err := block.DeleteDownToBlock(startingPoint)
						if nil != err {
							log.Errorf("delete down to block number: %d  error: %s", startingPoint, err)
						}

						m.fastsyncEnabled = false
						m.nextState(cStateHighestBlock)
						m.startBlockNumber = startingPoint
						break fetchBlocks
					}
				}

				// get next block
				//   packedNextBlock will be nil when local height is same as remote
				var err error
				packedNextBlock, err = m.attachedNode.GetBlockData(m.targetNode.(*P2PCandidatesImpl).ID, m.startBlockNumber+1, nil, nil)
				if nil != err {
					log.Debugf("fetch block number: %d  error: %s", m.startBlockNumber+1, err)
				}
			} else {
				packedNextBlock = nil
			}

			log.Debugf("store block number: %d", m.startBlockNumber)
			err := block.StoreIncoming(packedBlock, packedNextBlock, block.NoRescanVerified)
			if nil != err {
				log.Errorf(
					"store block number: %d  error: %s",
					m.startBlockNumber,
					err,
				)
				m.nextState(cStateHighestBlock) // retry
				break fetchBlocks
			}

			// next block
			m.startBlockNumber++
		}

	case cStateRebuild:
		util.LogInfo(log, util.CoYellow, fmt.Sprintf("Enter Rebuild state, mode:%s", mode.String()))
		// return to normal operations
		m.nextState(cStateSampling)
		m.samples = 0 // zero out the counter
		mode.Set(mode.Normal)
		stop = true

	case cStateSampling:
		util.LogInfo(log, util.CoYellow, fmt.Sprintf("Enter Sampling state, mode:%s", mode.String()))
		// check peers
		connCount := m.attachedNode.MetricsNetwork.GetConnCount()
		if !isConnectionEnough(connCount) {
			log.Warnf("connections: %d below minimum client count: %d", connCount, minimumClients)
			stop = false
			m.nextState(cStateConnecting)
			return stop

		}

		log.Infof("connections: %d", connCount)
		stop = true

		// check height
		if m.updateHeightAndClient() {
			height := blockheader.Height()

			log.Infof("height remote: %d, local: %d", m.targetHeight, height)

			if m.hasBetterChain(height) {
				log.Warn("check height: better chain")
				m.nextState(cStateForkDetect)
				stop = false
			} else {
				m.samples = 0
			}
		} else {
			m.samples++
			if m.samples > samplingLimit {
				log.Warn("check height: time to resync")
				m.nextState(cStateForkDetect)
				stop = false
			}
		}
	}
	return stop
}

func (m *Machine) nextState(newState state) {
	m.state = newState
}

func isConnectionEnough(count counter.Counter) bool {
	return minimumClients <= int64(count)
}

func (m *Machine) startElection() {
	m.votingMetrics.allCandidates(func(c *P2PCandidatesImpl) {
		if c.ActiveInThePast(activeInterval) {
			m.votes.VoteBy(c)
		}
	})
}

func (m *Machine) updateHeightAndClient() bool {
	m.votes.Reset()
	m.votes.SetMinHeight(blockheader.Height())
	m.startElection()
	elected, height := m.elected()
	if 0 == height {
		m.targetHeight = 0
		return false
	}

	name := elected.Name()
	addr := elected.RemoteAddr()
	if addr == "" {
		m.log.Warnf("%s socket not connected", name)
		m.targetHeight = 0
		return false
	}

	m.log.Debugf("winner %s majority height %d, connect to %s",
		name,
		height,
		addr,
	)

	m.targetNode = elected
	m.targetHeight = height
	return true
}

func (m *Machine) elected() (voting.Candidate, uint64) {
	elected, height, err := m.votes.ElectedCandidate()
	if nil != err {
		m.log.Errorf("get elected with error: %s", err.Error())
		return nil, 0
	}

	remoteAddr := elected.RemoteAddr()
	if remoteAddr == "" {
		m.log.Errorf("get client string with error: %s", err)
		return nil, 0
	}

	digest := elected.CachedRemoteDigestOfLocalHeight()
	util.LogDebug(m.log, util.CoReset, fmt.Sprintf(
		"digest: %s elected with %d votes, remote addr: %s, height: %d",
		digest,
		m.votes.NumVoteOfDigest(digest),
		remoteAddr,
		height,
	))

	return elected, height
}

func (m *Machine) hasBetterChain(localHeight uint64) bool {
	if m.targetNode == nil || m.targetNode.Name() == "" {
		m.log.Debug("remote client empty")
		return false
	}

	if m.targetHeight < localHeight {
		m.log.Debugf("remote height %d is shorter than local height %d", m.targetHeight, localHeight)
		return false
	}

	if m.targetHeight == localHeight && !m.hasSmallerDigestThanLocal(localHeight) {
		return false
	}

	return true
}

// different chain but with same height, possible fork exist
// choose the chain that has smaller digest
func (m *Machine) hasSmallerDigestThanLocal(localHeight uint64) bool {
	remoteDigest := m.targetNode.CachedRemoteDigestOfLocalHeight()

	// if remote node updates during state machine flow
	if m.targetNode.(*P2PCandidatesImpl).Metrics.localHeight != localHeight {
		m.log.Warnf("remote height %d is different than local height %d", m.targetNode.(*P2PCandidatesImpl).Metrics.localHeight, localHeight)
		return false
	}

	localDigest, err := blockheader.DigestForBlock(localHeight)
	if nil != err {
		m.log.Warnf("local height: %d  digest error: %s", localHeight, err)
		return false
	}

	return remoteDigest.SmallerDigestThan(localDigest)
}

func (m *Machine) isSameChain() bool {
	if m.targetNode == nil || m.targetNode.Name() == "" {
		m.log.Debug("remote node empty")
		return false
	}

	localDigest, err := blockheader.DigestForBlock(blockheader.Height())
	if nil != err {
		return false
	}

	if m.targetHeight == blockheader.Height() && m.targetNode.CachedRemoteDigestOfLocalHeight() == localDigest {
		return true
	}

	return false
}
