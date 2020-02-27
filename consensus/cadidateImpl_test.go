package consensus

import (
	cryptorand "crypto/rand"
	"fmt"
	"math"
	mathrand "math/rand"
	"strconv"
	"testing"
	"time"

	crypto "github.com/libp2p/go-libp2p-core/crypto"
	peerlib "github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/merkle"
)

func TestCachedRemoteHeight(t *testing.T) {
	candidates, err := mockCadidates(30)
	assert.NoError(t, err, "gen mockdata error")
	for _, candidate := range candidates {
		target := candidate.Metrics.remoteHeight
		actual := candidate.CachedRemoteHeight()
		assert.Equal(t, target, actual, fmt.Sprintf("CachedRemoteHeight target:%d, actual:%d", target, actual))
	}
}
func TestCachedRemoteDigestOfLocalHeight(t *testing.T) {
	candidates, err := mockCadidates(30)
	assert.NoError(t, err, "gen mockdata error")
	for _, candidate := range candidates {
		target := candidate.Metrics.remoteDigestOfLocalHeight
		actual := candidate.CachedRemoteDigestOfLocalHeight()
		assert.Equal(t, target, actual, fmt.Sprintf("CachedRemoteDigestOfLocalHeight target:%d, actual:%d", target, actual))
	}
}

func TestRemoteAddr(t *testing.T) {
	candidates, err := mockCadidates(30)
	assert.NoError(t, err, "gen mockdata error")
	for _, candidate := range candidates {
		target := candidate.Addr.String()
		actual := candidate.RemoteAddr()
		assert.EqualValues(t, target, actual, fmt.Sprintf("RemoteAddr target:%s, actual:%s", target, actual))
		candidate.Addr = nil
		target = ""
		actual = candidate.RemoteAddr()
		assert.EqualValues(t, target, actual, fmt.Sprintf("RemoteAddr target: empty string, actual:%s", actual))

	}
}
func TestName(t *testing.T) {
	candidates, err := mockCadidates(30)
	assert.NoError(t, err, "gen mockdata error")
	for _, candidate := range candidates {
		target := candidate.ID.Pretty()
		actual := candidate.Name()
		assert.EqualValues(t, target, actual, fmt.Sprintf("RemoteAddr target:%s, actual:%s", target, actual))
	}
}
func TestActiveInThePast(t *testing.T) {
	candidates, err := mockCadidates(1)
	assert.NoError(t, err, "gen mockdata error")
	time.Sleep(2 * time.Second)
	active := candidates[0].ActiveInThePast(1 * time.Second)
	assert.Equal(t, false, active, "should pass active second but not")
}
func TestSetMetrics(t *testing.T) {
	candidates, err := mockCadidates(30)
	assert.NoError(t, err, "gen mockdata error")
	i := 0
	for _, c := range candidates {
		idx := math.Mod(float64(i), 31)
		newName := c.Name()
		newRemoteHeight := c.Metrics.remoteHeight + uint64(i)
		newLocalHeight := c.Metrics.localHeight + uint64(i)
		c.Metrics.remoteDigestOfLocalHeight[int(idx)] = byte(3)
		newDigest := blockdigest.Digest(c.Metrics.remoteDigestOfLocalHeight)
		newTime := time.Now()

		c.UpdateMetrics(newName, newRemoteHeight, newLocalHeight, newDigest, newTime)
		assert.Equal(t, newName, c.Name(), fmt.Sprintf("Name target:%s, actual:%s", newName, c.Name()))
		assert.Equal(t, newRemoteHeight, c.CachedRemoteHeight(), fmt.Sprintf("CachedRemoteHeight target: %d, actual:%d", newRemoteHeight, c.CachedRemoteHeight()))
		assert.Equal(t, newLocalHeight, c.Metrics.localHeight, fmt.Sprintf("localHeight target:%d, actual:%d", newLocalHeight, c.Metrics.localHeight))
		assert.Equal(t, newDigest, c.CachedRemoteDigestOfLocalHeight(), fmt.Sprintf("localHeight target:%d, actual:%d", newDigest, c.CachedRemoteDigestOfLocalHeight()))
		assert.Equal(t, newTime, c.Metrics.lastResponseTime, fmt.Sprintf("localHeight target:%d, actual:%d", newTime.Unix(), c.Metrics.lastResponseTime.Unix()))
		i++
	}
}

func mockCadidates(n int) ([]P2PCandidatesImpl, error) {
	candidates := []P2PCandidatesImpl{}
	//randInc := []byte(strconv.Itoa(mathrand.Intn(3)))
	randInc := mathrand.Intn(3)

	for i := 0; i < n; i++ {
		randID, err := genID()
		if err != nil {
			return nil, err
		}
		digest, err := genDigest()
		if err != nil {
			return nil, err
		}
		votingData := metricsVoting{
			name:                      randID.Pretty(),
			remoteHeight:              10 + uint64(randInc),
			localHeight:               11,
			remoteDigestOfLocalHeight: digest,
		}
		addr, err := ma.NewMultiaddr("/ip4/127.0.0.1/udp/1234")
		if err != nil {
			return nil, err
		}
		candidate := P2PCandidatesImpl{
			ID:      peerlib.ID(votingData.name),
			Metrics: votingData,
			Addr:    addr,
		}
		candidates = append(candidates, candidate)
	}

	return candidates, nil
}

func genDigest() ([32]byte, error) {
	data := []byte(strconv.Itoa(mathrand.Intn(1000000000000000)))
	return merkle.NewDigest(data[:]), nil
}

func genID() (peerlib.ID, error) {
	privKey, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, 0, cryptorand.Reader)
	if err != nil {
		return peerlib.ID(""), err
	}
	libp2pID, err := peerlib.IDFromPrivateKey(privKey)
	if err != nil {
		return peerlib.ID(""), err
	}
	return libp2pID, nil
}
