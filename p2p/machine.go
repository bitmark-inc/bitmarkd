package p2p

import (
	"fmt"
	"time"

	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/blockheader"
	"github.com/bitmark-inc/bitmarkd/counter"
	"github.com/bitmark-inc/bitmarkd/genesis"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/p2p/voting"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/logger"
)

// various timeouts
const (
	// pause to limit bandwidth
	cycleInterval = 15 * time.Second

	// number of cycles to be 1 block out of sync before resync
	samplelingLimit = 10

	// number of blocks to fetch in one set
	fetchBlocksPerCycle = 200

	// fail to fork if height difference is greater than this
	forkProtection = 60

	// do not proceed unless this many clients are connected
	minimumClients = 5

	// total number of dynamic clients
	maximumDynamicClients = 25

	// client should exist at least 1 response with in this number
	activePastSec = 60
)

// Machine voting concensus state machine
type Machine struct {
	log *logger.L
	state
	attachedNode     *Node
	votingMetrics    *MetricsPeersVoting
	votes            voting.Voting
	electedWiner     voting.Candidate //voting winner
	electedHeight    uint64           //voting winner block height
	startBlockNumber uint64
	samples          int
}

// NewConcensusMachine get a new StateMachine
func NewConcensusMachine(node *Node, metric *MetricsPeersVoting) Machine {
	machine := Machine{log: logger.New("concensus"), votingMetrics: metric, attachedNode: node}
	machine.toState(cStateConnecting)
	machine.votes = voting.NewVoting()
	return machine
}

//Run Run A ConcensusMachine
func (m *Machine) Run(args interface{}, shutdown <-chan struct{}) {
	log := m.log
	log.Info("starting a concensus state machine…")
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
			m.nextState()
		} else {
			log.Debugf("connections: %d below minimum client count: %d", uint64(m.attachedNode.MetricsNetwork.GetConnCount()), minimumClients)
		}
		stop = true
	case cStateHighestBlock:
		util.LogInfo(log, util.CoYellow, fmt.Sprintf("Enter HighestBlock state, mode:%s", mode.String()))
		winerHeight, winer := m.newElection()
		if winer == nil || 0 == winerHeight {
			stop = true
		} else {
			m.electedHeight = winerHeight
			m.electedWiner = winer
			if m.hasBetterChain(blockheader.Height()) {
				util.LogInfo(log, util.CoReset, fmt.Sprintf("new chain from %s, height %d, digest %s",
					m.electedWiner.Name(), m.electedWiner.CachedRemoteHeight(), m.electedWiner.CachedRemoteDigestOfLocalHeight().String()))
				m.nextState()
			} else if m.isSameChain() {
				m.toState(cStateRebuild)
			} else {
				util.LogError(log, util.CoRed, fmt.Sprintf("remote chain invalid, stop looping for now"))
				stop = true
			}
		}
	case cStateForkDetect:
		util.LogInfo(log, util.CoYellow, fmt.Sprintf("Enter ForkDetect state, mode:%s", mode.String()))
		height := blockheader.Height()
		if !m.hasBetterChain(height) {
			m.toState(cStateRebuild)
		} else {
			mode.Set(mode.Resynchronise)
			// first block number
			m.startBlockNumber = genesis.BlockNumber + 1
			m.nextState() // assume success
			log.Infof("local block number: %d", height)

			blockheader.ClearCache()
			// check digests of descending blocks (to detect a fork)
		check_digests:
			for h := height; h >= genesis.BlockNumber; h -= 1 {
				digest, err := blockheader.DigestForBlock(h)
				if nil != err {
					log.Infof("block number: %d  local digest error: %s", h, err)
					m.toState(cStateHighestBlock) // retry
					break check_digests
				}
				d, err := m.attachedNode.RemoteDigestOfHeight(m.electedWiner.(*P2PCandidatesImpl).ID, h, nil, nil)
				if nil != err {
					log.Infof("block number: %d  fetch digest error: %s", h, err)
					m.toState(cStateHighestBlock) // retry
					break check_digests
				} else if d == digest {
					if height-h >= forkProtection {
						log.Errorf("fork protection at: %d - %d >= %d", height, h, forkProtection)
						m.toState(cStateHighestBlock)
						break check_digests
					}

					m.startBlockNumber = h + 1
					log.Infof("fork from block number: %d", m.startBlockNumber)
					// remove old blocks
					err := block.DeleteDownToBlock(m.startBlockNumber)
					if nil != err {
						log.Errorf("delete down to block number: %d  error: %s", m.startBlockNumber, err)
						m.toState(cStateHighestBlock) // retry
					}
					break check_digests
				}
			}
		}
	case cStateFetchBlocks:
		util.LogInfo(log, util.CoYellow, fmt.Sprintf("Enter FetchBlocks state, mode:%s", mode.String()))
		stop = true
	fetch_blocks:
		for n := 0; n < fetchBlocksPerCycle; n++ {
			if m.startBlockNumber > m.electedHeight {
				// just in case block height has changed
				m.toState(cStateHighestBlock)
				stop = false
				break fetch_blocks
			}
			packedBlock, err := m.attachedNode.GetBlockData(m.electedWiner.(*P2PCandidatesImpl).ID, m.startBlockNumber, nil, nil)
			if nil != err {
				log.Errorf("fetch block number: %d  error: %s", m.startBlockNumber, err)
				m.toState(cStateHighestBlock) // retry
				break fetch_blocks
			}
			err = block.StoreIncoming(packedBlock, nil, block.NoRescanVerified)
			if nil != err {
				util.LogInfo(log, util.CoRed, fmt.Sprintf(
					"store block number: %d  error: %s",
					m.startBlockNumber,
					err,
				))
				m.toState(cStateHighestBlock) // retry
				break fetch_blocks
			}

			// next block
			m.startBlockNumber++

		}
	case cStateRebuild:
		util.LogInfo(log, util.CoYellow, fmt.Sprintf("Enter Rebuild state, mode:%s", mode.String()))
		// return to normal operations
		m.nextState()
		m.samples = 0 // zero out the counter
		mode.Set(mode.Normal)
		stop = true
	case cStateSampling:
		util.LogInfo(log, util.CoYellow, fmt.Sprintf("Enter Sampling state, mode:%s", mode.String()))
		// check peers
		//globalData.clientCount = conn.getConnectedClientCount()
		connCount := m.attachedNode.MetricsNetwork.GetConnCount()
		if !isConnectionEnough(connCount) {
			stop = false
			m.toState(cStateConnecting)
			return stop
		}
		// check height
		winerHeight, winer := m.newElection()
		m.electedHeight = winerHeight
		m.electedWiner = winer
		height := blockheader.Height()
		util.LogInfo(log, util.CoWhite, fmt.Sprintf("height remote: %d, local: %d", m.electedHeight, height))
		stop = true
		if m.hasBetterChain(height) {
			m.toState(cStateForkDetect)
			stop = false
		} else {
			m.samples++
			if m.samples > samplelingLimit {
				m.toState(cStateForkDetect)
				stop = false
			}
		}
	}
	return stop
}

func (m *Machine) toState(newState state) {
	m.state = newState
}
func (m *Machine) nextState() {
	m.state++
}

func isConnectionEnough(count counter.Counter) bool {
	return minimumClients <= int64(count)
}

func (m *Machine) startElection() {
	candidatNum := 0
	voteCandidatNum := 0
	m.votingMetrics.allCandidates(func(c *P2PCandidatesImpl) {
		candidatNum++
		if c.ActiveInPastSeconds(activePastSec) {
			m.votes.VoteBy(c)
			voteCandidatNum++
		}
	})
}

func (m *Machine) newElection() (uint64, voting.Candidate) {
	m.votes.Reset()
	m.votes.SetMinHeight(blockheader.Height())
	m.startElection()
	elected, height := m.elected()
	if uint64(0) == height {
		return uint64(0), nil
	}
	winnerName := elected.Name()
	remoteAddr := elected.RemoteAddr()
	util.LogDebug(m.log, util.CoWhite, fmt.Sprintf("winner %s majority height %d, connect to %s",
		winnerName,
		height,
		remoteAddr,
	))
	if height > uint64(0) && nil != elected {
		m.electedHeight = height
	}
	return height, elected
}

func (m *Machine) elected() (voting.Candidate, uint64) {
	elected, height, err := m.votes.ElectedCandidate()
	if nil != err {
		m.log.Errorf("get elected with error: %s", err.Error())
		return nil, uint64(0)
	}

	remoteAddr := elected.RemoteAddr()
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
	if m.electedHeight < localHeight {
		m.log.Debugf("remote height %d is shorter than local height %d", m.electedHeight, localHeight)
		return false
	}
	if m.electedHeight == localHeight && !m.hasSamllerDigestThanLocal(localHeight) {
		return false
	}
	return true
}

// different chain but with same height, possible fork exist
// choose the chain that has smaller digest
func (m *Machine) hasSamllerDigestThanLocal(localHeight uint64) bool {
	remoteDigest := m.electedWiner.CachedRemoteDigestOfLocalHeight()
	// if upstream update during processing
	if m.electedWiner.(*P2PCandidatesImpl).Metrics.localHeight != localHeight {
		m.log.Warnf("remote height %d is different than local height %d", m.electedWiner.(*P2PCandidatesImpl).Metrics.localHeight, localHeight)
		return false
	}
	localDigest, err := blockheader.DigestForBlock(localHeight)
	if nil != err {
		return false
	}
	return remoteDigest.SmallerDigestThan(localDigest)
}

func (m *Machine) isSameChain() bool {
	if m.electedWiner.Name() == "" {
		m.log.Debug("no winner")
		return false
	}
	localDigest, err := blockheader.DigestForBlock(blockheader.Height())
	if nil != err {
		return false
	}
	if m.electedHeight == blockheader.Height() && m.electedWiner.CachedRemoteDigestOfLocalHeight() == localDigest {
		return true
	}
	return false
}
