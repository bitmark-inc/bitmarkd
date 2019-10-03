package p2p

import (
	"time"

	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/libp2p/go-libp2p-core/peer"
)

type votingMetricsIntf interface {
	CachedRemoteDigestOfLocalHeight() blockdigest.Digest
	ClientAddr() string
	Name() string
}

type votingMetrics struct {
	name                      peer.ID
	remoteHeight              uint64
	localHeight               uint64
	remoteDigestOfLocalHeight blockdigest.Digest
	lastResponseTime          time.Time
}

func (v *votingMetrics) CachedRemoteDigestOfLocalHeight() blockdigest.Digest {
	return blockdigest.Digest{}
}
func (v *votingMetrics) ClientAddr() string {
	return ""
}
func (v *votingMetrics) Name() string {
	return ""
}
