package p2p

import (
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	crypto "github.com/libp2p/go-libp2p-core/crypto"
	peerlib "github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/assert"
)

const (
	registersNum   = 10
	expireDuration = 1 * time.Second // must be greater than 10 microseconds
)

func TestNewRegistration(t *testing.T) {
	_, ids, err := newMockRegistrationData()
	assert.NoError(t, err, fmt.Sprintf("generate mock data error:%v", err))
	assert.Equal(t, len(ids), registersNum, fmt.Sprintf("mock data generate number is wrong:"))

}

func TestAddRegistration(t *testing.T) {
	reg, ids, err := newMockRegistrationData()
	assert.NoError(t, err, fmt.Sprintf("generate mock data error:%v", err))
	for _, id := range ids {
		reg.addRegister(id)
		val, ok := reg.peers[id]
		assert.Equal(t, true, ok, "addRegister does not write peer id ")
		time.Sleep(5 * time.Nanosecond)
		assert.Equal(t, true, val.Registered, "addRegister does not write correct status")
		assert.Greater(t, time.Now().UnixNano(), val.LatestRegisterTime.UnixNano(), fmt.Sprintf("time is not greater"))
	}
}

func TestUnRegistered(t *testing.T) {
	reg, ids, err := newMockRegistrationData()
	assert.NoError(t, err, fmt.Sprintf("generate mock data error:%v", err))
	for _, id := range ids {
		reg.addRegister(id)
	}
	id1 := ids[len(ids)/2]
	id2 := ids[len(ids)/4]
	reg.unRegister(id1)
	reg.unRegister(id2)
	status, ok := reg.peers[id1]
	assert.Equal(t, true, ok, fmt.Sprintf("unRegister  peer not in the list ok=%v", ok))
	assert.Equal(t, false, status.Registered, fmt.Sprintf("unRegister  peer fail status=%v id=%v", status.Registered, id1))
	status, ok = reg.peers[id2]
	assert.Equal(t, true, ok, fmt.Sprintf("unRegister  peer not in the list"))
	assert.Equal(t, false, status.Registered, fmt.Sprintf("unRegister  peer fail"))
}

func TestRegisteredPeers(t *testing.T) {
	reg, ids, err := newMockRegistrationData()
	assert.NoError(t, err, fmt.Sprintf("generate mock data error:%v", err))
	for _, id := range ids {
		reg.addRegister(id)
	}
	regPeers := reg.RegisteredPeers()
	for _, id := range regPeers {
		_, ok := reg.peers[id]
		assert.Equal(t, true, ok, "id is not in the registreredPeers")
	}
}
func TestIsRegistered(t *testing.T) {
	reg, ids, err := newMockRegistrationData()
	assert.NoError(t, err, fmt.Sprintf("generate mock data error:%v", err))

	id1 := ids[len(ids)/2]
	id2 := ids[len(ids)/4]
	reg.addRegister(id1)
	status, ok := reg.peers[id1]
	assert.Equal(t, true, ok, fmt.Sprintf("peer is not in the peers list"))
	assert.Equal(t, true, reg.IsRegistered(id1), fmt.Sprintf("status should be %v but return %v", status.Registered, reg.IsRegistered(id1)))
	status, ok = reg.peers[id2]
	assert.Equal(t, true, ok, fmt.Sprintf("peer is not in the peers list"))
	assert.Equal(t, false, reg.IsRegistered(id2), fmt.Sprintf("status should be %v but return %v", status.Registered, reg.IsRegistered(id2)))

}
func newMockRegistrationData() (*BasicPeerRegistration, []peerlib.ID, error) {
	ids := []peerlib.ID{}
	registration := NewRegistration(expireDuration)
	for i := 0; i < registersNum; i++ {
		privKey, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, 0, rand.Reader)
		if err != nil {
			return nil, nil, err
		}
		libp2pID, err := peerlib.IDFromPrivateKey(privKey)
		if err != nil {
			return nil, nil, err
		}
		registration.(*BasicPeerRegistration).peers[libp2pID] = &RegistrationStatus{Registered: false}
		ids = append(ids, libp2pID)
	}
	return registration.(*BasicPeerRegistration), ids, nil
}
