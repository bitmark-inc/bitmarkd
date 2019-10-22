package p2p

import (
	"time"

	"github.com/bitmark-inc/bitmarkd/blockdigest"
	peerlib "github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

type metricsVoting struct {
	name                      string
	remoteHeight              uint64
	localHeight               uint64
	remoteDigestOfLocalHeight blockdigest.Digest
	lastResponseTime          time.Time
}

//P2PCandidatesImpl libp2p implementation of Candidate
type P2PCandidatesImpl struct {
	ID      peerlib.ID
	Addr    ma.Multiaddr
	Metrics metricsVoting
}

//CachedRemoteHeight return remote height
func (p *P2PCandidatesImpl) CachedRemoteHeight() uint64 {
	return p.Metrics.remoteHeight
}

//CachedRemoteDigestOfLocalHeight return local height
func (p *P2PCandidatesImpl) CachedRemoteDigestOfLocalHeight() blockdigest.Digest {
	return p.Metrics.remoteDigestOfLocalHeight
}

//RemoteAddr return remote height
func (p *P2PCandidatesImpl) RemoteAddr() string {
	if p.Addr != nil {
		return p.Addr.String()
	}
	return ""
}

//Name return name of cadidate
func (p *P2PCandidatesImpl) Name() string {
	return p.ID.String()
}

// ActiveInPastSeconds - active metrics in past seconds
func (p *P2PCandidatesImpl) ActiveInPastSeconds(sec time.Duration) bool {
	now := time.Now()
	limit := now.Add(time.Second * sec * -1)
	active := limit.Before(p.Metrics.lastResponseTime)
	difference := now.Sub(p.Metrics.lastResponseTime).Seconds()
	globalData.concensusMachine.log.Debugf("\x1b[33mActiveInPastSeconds active: %t, last response time %s, difference %f seconds\x1b[0m",
		active,
		p.Metrics.lastResponseTime.Format("2006-01-02, 15:04:05 -0700"),
		difference,
	)
	return active
}

//UpdateMetrics update metrics values
func (p *P2PCandidatesImpl) UpdateMetrics(name string, remoteHeight, localHeight uint64, digest blockdigest.Digest, respTime time.Time) {
	p.Metrics.name = name
	p.Metrics.remoteHeight = remoteHeight
	p.Metrics.localHeight = localHeight
	p.Metrics.remoteDigestOfLocalHeight = digest
	p.Metrics.lastResponseTime = respTime
}
