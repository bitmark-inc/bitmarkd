package p2p

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/bitmark-inc/bitmarkd/blockheader"

	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/logger"
	peerlib "github.com/libp2p/go-libp2p-core/peer"
)

const (
	votingCycleInterval = 30 * time.Second
	votingQueryTimeout  = 5 * time.Second
)

//MetricsPeersVoting  is to get all metrics for voting
type MetricsPeersVoting struct {
	mutex      *sync.Mutex
	watchNode  *Node
	Candidates []*P2PCandidatesImpl
	Log        *logger.L
}

//NewMetricsPeersVoting return a MetricsPeersVoting for voting
func NewMetricsPeersVoting(thisNode *Node) MetricsPeersVoting {
	var mutex = &sync.Mutex{}
	metrics := MetricsPeersVoting{mutex: mutex, watchNode: thisNode, Log: logger.New("votingMetrics")}
	metrics.UpdateCandidates()
	return metrics
}

//UpdateCandidates update Candidate by registered peer
func (p *MetricsPeersVoting) UpdateCandidates() {
	var Candidates []*P2PCandidatesImpl
	p.mutex.Lock()
	for peerID, status := range p.watchNode.Registers {
		if !util.IDEqual(p.watchNode.Host.ID(), peerID) {
			if status.Registered { // register and not self
				peerInfo := p.watchNode.Host.Peerstore().PeerInfo(peerID)
				if len(peerInfo.Addrs) > 0 {
					Candidates = append(Candidates, &P2PCandidatesImpl{ID: peerID, Addr: peerInfo.Addrs[0]})
				} else {
					Candidates = append(Candidates, &P2PCandidatesImpl{ID: peerID})
				}
			}
		}

	}
	p.Candidates = Candidates
	p.mutex.Unlock()
	util.LogInfo(p.Log, util.CoWhite, fmt.Sprintf("@@UpdateCandidates:%d Candidates!", len(Candidates)))
}

//Run  is a Routine to get peer info
func (p *MetricsPeersVoting) Run(args interface{}, shutdown <-chan struct{}) {
	log := p.Log
	delay := time.After(nodeInitial)
	//nodeChain:= mode.ChainName()
	util.LogWarn(log, util.CoReset, "MetricsPeersVoting routine start...")
loop:
	for {
		select {
		case <-shutdown:
			continue loop
		case <-delay: //update voting metrics
			//util.LogWarn(log, util.CoCyan, "MetricsPeersVoting  routine interval...")
			delay = time.After(votingCycleInterval)
			p.UpdateCandidates()
			if nil == p.Candidates {
				util.LogInfo(p.Log, util.CoRed, "Candidates: no Candidates")
				continue loop
			}
			for _, peer := range p.Candidates {
				go func(id peerlib.ID) {
					err := p.watchNode.UpdateVotingMetrics(id, p)
					if err != nil {
						util.LogWarn(p.Log, util.CoRed, fmt.Sprintf("UpdateVotingMetrics Error:%v", err))
					}
				}(peer.ID)
			}
		}
	}
}

//SetMetrics set the voting metrics value
func (p *MetricsPeersVoting) SetMetrics(id peerlib.ID, height uint64, digest blockdigest.Digest) {
	for _, candidate := range p.Candidates {
		if util.IDEqual(candidate.ID, id) {
			localheight := blockheader.Height()
			respTime := time.Now()
			p.mutex.Lock()
			candidate.UpdateMetrics(id.String(), height, localheight, digest, respTime)
			p.mutex.Unlock()
			util.LogInfo(p.Log, util.CoReset, fmt.Sprintf("SetMetrics:ID:%s, remoteHeight:%d, localHeight:%d, digest:%s, responseTime:%v", id.ShortString(), height, localheight, digest, respTime))
			break
		}
	}
}

//GetMetrics get the voting metrics value
func (p *MetricsPeersVoting) GetMetrics(id peerlib.ID) (name string, remoteHeight uint64, localHeight uint64, digest blockdigest.Digest, lastRespTime time.Time, err error) {
	for _, candidate := range p.Candidates {
		if util.IDEqual(candidate.ID, id) {
			return candidate.Metrics.name, candidate.Metrics.remoteHeight, candidate.Metrics.localHeight, candidate.Metrics.remoteDigestOfLocalHeight, candidate.Metrics.lastResponseTime, nil
		}
	}
	return "", 0, 0, blockdigest.Digest{}, time.Time{}, errors.New("No Peer ID")
}

func (p *MetricsPeersVoting) allCandidates(
	f func(candidate *P2PCandidatesImpl),
) {
	for _, candidate := range p.Candidates {
		f(candidate)
	}
}
