package p2p

import (
	"sync"
	"time"

	"github.com/bitmark-inc/bitmarkd/blockheader"

	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/logger"
	peerlib "github.com/libp2p/go-libp2p-core/peer"
)

const (
	votingCycleInterval = 10 * time.Second
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

//UpdateCandidates update Candidate by current peer
func (p *MetricsPeersVoting) UpdateCandidates() {
	var Candidates []*P2PCandidatesImpl
	var candidate P2PCandidatesImpl
	peerstore := p.watchNode.Host.Peerstore()
	p.mutex.Lock()
	for _, id := range peerstore.PeersWithAddrs() {
		if p.watchNode.IsRegister(id) && !util.IDEqual(p.watchNode.Host.ID(), id) {
			candidate = P2PCandidatesImpl{ID: id}
			addrs := peerstore.Addrs(id)
			if len(addrs) > 1 {
				candidate.Addr = addrs[0]
				if p.watchNode.PreferIPv6 {
					for _, addr := range addrs {
						if util.IsMultiAddrIPV6(addr) {
							candidate.Addr = addr
							break
						}
					}
				}
			}
		}
		Candidates = append(Candidates, &candidate)
	}
	p.Candidates = Candidates
	p.mutex.Unlock()
	for _, c := range p.Candidates {
		p.Log.Debugf("Current candidate ID :%s addr:%v", c.ID, c.Addr)
	}

}

//Run  is a Routine to get peer info
func (p *MetricsPeersVoting) Run(args interface{}, shutdown <-chan struct{}) {
	log := p.Log
	delay := time.After(nodeInitial)
	//nodeChain:= mode.ChainName()
loop:
	for {
		log.Debug("waiting…")
		select {
		case <-shutdown:
			break loop
		case <-delay: //update voting metrics
			delay = time.After(votingCycleInterval)
			p.UpdateCandidates()
			for _, peer := range p.Candidates {
				go func(id peerlib.ID) {
					height, err := p.watchNode.QueryBlockHeight(peer.ID)
					if err != nil {
						p.Log.Errorf("\x1b[31mRun QueryBlockHeight Error : %v\x1b[0m", err)
						return
					}
					digest, err := p.watchNode.RemoteDigestOfHeight(id, height)
					if err != nil {
						p.Log.Errorf("\x1b[31mRun RemoteDigestOfHeight Error : %v\x1b[0m", err)
						return
					}
					p.Log.Debugf("\x1b[33mID Query Return height: %d candidates\x1b[0m", height)
					p.setMetrics(id, height, digest)
				}(peer.ID)
			}
		}
	}
}

func (p *MetricsPeersVoting) setMetrics(id peerlib.ID, height uint64, digest blockdigest.Digest) {
	for _, candidate := range p.Candidates {
		if util.IDEqual(candidate.ID, id) {
			p.mutex.Lock()
			localheight := blockheader.Height()
			respTime := time.Now()
			candidate.UpdateMetrics(id.String(), height, localheight, digest, respTime)
			p.mutex.Unlock()
			p.Log.Debugf("\x1b[32m: ID:%s, height:%d, digest:%s respTime:%d\x1b[0m",
				candidate.ID, candidate.Metrics.localHeight, candidate.Metrics.remoteDigestOfLocalHeight, candidate.Metrics.lastResponseTime.Unix())
			break
		}
	}
}

func (p *MetricsPeersVoting) allCandidates(
	f func(candidate *P2PCandidatesImpl),
) {
	for _, candidate := range p.Candidates {
		f(candidate)
	}
}
