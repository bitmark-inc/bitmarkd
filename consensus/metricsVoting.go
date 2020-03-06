package consensus

import (
	"bufio"
	"context"
	"fmt"
	"sync"
	"time"

	peerlib "github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"

	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/blockheader"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/p2p"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/logger"
)

const (
	waitingRespTime = 30 * time.Second
)

//MetricsPeersVoting  is to get all metrics for voting
type MetricsPeersVoting struct {
	sync.Mutex
	watchNode  *p2p.Node
	Candidates []*P2PCandidatesImpl
	Log        *logger.L
}

//NewMetricsPeersVoting return a MetricsPeersVoting for voting
func NewMetricsPeersVoting(thisNode *p2p.Node) *MetricsPeersVoting {
	metrics := &MetricsPeersVoting{watchNode: thisNode, Log: globalData.Log}
	metrics.UpdateCandidates()
	return metrics
}

//UpdateCandidates update Candidate by registered peer
func (p *MetricsPeersVoting) UpdateCandidates() {
	var Candidates []*P2PCandidatesImpl
	p.Lock()
	for _, peerID := range p.watchNode.Registers.RegisteredPeers() {
		if !util.IDEqual(p.watchNode.Host.ID(), peerID) {
			peerInfo := p.watchNode.Host.Peerstore().PeerInfo(peerID)
			if len(peerInfo.Addrs) > 0 {
				Candidates = append(Candidates, &P2PCandidatesImpl{ID: peerID, Addr: peerInfo.Addrs[0]})
			} else {
				Candidates = append(Candidates, &P2PCandidatesImpl{ID: peerID})
			}
		}

	}
	p.Candidates = Candidates
	p.Unlock()
	util.LogInfo(p.Log, util.CoWhite, fmt.Sprintf("@@UpdateCandidates:%d Candidates!", len(Candidates)))
}

//UpdateVotingMetrics Register first and get info for voting metrics. This is an  efficient way to get data without create a new stream
func (p *MetricsPeersVoting) UpdateVotingMetrics(id peerlib.ID) error {
	cctx, cancel := context.WithTimeout(context.Background(), waitingRespTime)
	defer cancel()
	s, err := p.watchNode.Host.NewStream(cctx, id, protocol.ID(p2p.TopicP2P))
	if err != nil {
		util.LogWarn(p.Log, util.CoRed, fmt.Sprintf("UpdateVotingMetrics: Create new stream for ID:%v Error:%v", id.ShortString(), err))
		return err
	}
	defer s.Close()
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
	_, err = p.watchNode.RequestRegister(id, s, rw)
	if err != nil {
		util.LogError(p.Log, util.CoRed, fmt.Sprintf("UpdateMetrics  RequestRegister Error:%v", err))
		return err
	}
	height, err := p.watchNode.QueryBlockHeight(id, s, rw)
	if err != nil {
		util.LogError(p.Log, util.CoRed, fmt.Sprintf("UpdateMetrics  QueryBlockHeight Error:%v", err))
		return err
	}
	digest, err := p.watchNode.RemoteDigestOfHeight(id, height, s, rw)
	if err != nil {
		util.LogError(p.Log, util.CoRed, fmt.Sprintf("UpdateMetrics  RemoteDigestOfHeight Error:%v", err))
		return err
	}
	p.SetMetrics(id, height, digest)
	return nil
}

//Run  is a Routine to get peer info
func (p *MetricsPeersVoting) Run(args interface{}, shutdown <-chan struct{}) {
	log := p.Log
	delay := time.After(votingMetricRunInitial)
	//nodeChain:= mode.ChainName()
	util.LogWarn(log, util.CoReset, "MetricsPeersVoting routine start...")
loop:
	for {
		select {
		case <-shutdown:
			continue loop
		case <-delay: //update voting metrics
			delay = time.After(votingMetricRunInterval)
			p.UpdateCandidates()
			if nil == p.Candidates {
				util.LogInfo(p.Log, util.CoRed, "Candidates: no Candidates")
				continue loop
			}
			for _, peer := range p.Candidates {
				go func(id peerlib.ID) {
					err := p.UpdateVotingMetrics(id)
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
loop:
	for _, candidate := range p.Candidates {
		if util.IDEqual(candidate.ID, id) {
			localheight := blockheader.Height()
			respTime := time.Now()
			p.Lock()
			candidate.UpdateMetrics(id.String(), height, localheight, digest, respTime)
			p.Unlock()
			//util.LogInfo(p.Log, util.CoCyan, fmt.Sprintf("SetMetrics:ID:%s, remoteHeight:%d, localHeight:%d, digest:%s, responseTime:%v", id.ShortString(), height, localheight, digest, respTime))
			break loop
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
	return "", 0, 0, blockdigest.Digest{}, time.Time{}, fault.NoPeerId
}

func (p *MetricsPeersVoting) allCandidates(
	f func(candidate *P2PCandidatesImpl),
) {
	for _, candidate := range p.Candidates {
		f(candidate)
	}
}
