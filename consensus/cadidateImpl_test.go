package consensus

import (
	cryptorand "crypto/rand"
	mathrand "math/rand"
	"strconv"
	"testing"

	"github.com/bitmark-inc/bitmarkd/merkle"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	peerlib "github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/stretchr/testify/assert"
)

func TestCachedRemoteHeight(t *testing.T) {
	_, err := mockCadidates(30)
	assert.NoError(t, err, "gen mockdata error")
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
		return candidates, nil
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
