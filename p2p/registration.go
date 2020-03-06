package p2p

import (
	"fmt"
	"sync"
	"time"

	peerlib "github.com/libp2p/go-libp2p-core/peer"

	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/logger"
)

// PeerRegisteration is an interface to register peers
type PeerRegisteration interface {
	RegisteredPeers() []peerlib.ID
	IsRegistered(id peerlib.ID) bool
	addRegister(id peerlib.ID)
	unRegister(id peerlib.ID)
	delRegister(id peerlib.ID)
	updateRegistersExpiry()
}

// BasicPeerRegistration a basic peer registration type
type BasicPeerRegistration struct {
	sync.RWMutex   // to allow locking
	Log            *logger.L
	peers          map[peerlib.ID]*RegistrationStatus
	expireDuration time.Duration
}

// RegistrationStatus is the struct to reflect the register status of a node
type RegistrationStatus struct {
	Registered         bool
	LatestRegisterTime time.Time
}

// NewRegistration return a BasicPeerRegistration entity
func NewRegistration(expireTime time.Duration) PeerRegisteration {
	return &BasicPeerRegistration{peers: make(map[peerlib.ID]*RegistrationStatus), expireDuration: expireTime, Log: globalData.Log}
}

// RegisteredPeers return current registered peers' ID
func (r *BasicPeerRegistration) RegisteredPeers() []peerlib.ID {
	r.RLock()
	defer r.RUnlock()
	peers := []peerlib.ID{}
	for id, status := range r.peers {
		if status.Registered {
			peers = append(peers, id)
		}
	}
	return peers
}

// IsRegistered return if the given ID peer registered
func (r *BasicPeerRegistration) IsRegistered(id peerlib.ID) bool {
	r.RLock()
	defer r.RUnlock()
	status := r.peers[id]
	return status != nil && status.Registered
}

func (r *BasicPeerRegistration) addRegister(id peerlib.ID) {
	r.Lock()
	r.peers[id] = &RegistrationStatus{Registered: true, LatestRegisterTime: time.Now()}
	r.Unlock()
	//util.LogInfo(r.Log, util.CoCyan, fmt.Sprintf("addRegister ID:%s Registered:%v time:%v", id.ShortString(), r.peers[id].Registered, r.peers[id].LatestRegisterTime.String()))
}

// unRegister unRegister change a peers's  Registered status  to false,  but it doe not not delete the register in the Registers
func (r *BasicPeerRegistration) unRegister(id peerlib.ID) {
	r.Lock()
	defer r.Unlock()
	status, ok := r.peers[id]
	if ok && status != nil { // keep LatestRegisterTime for last record purpose
		status.Registered = false
	}
}

// delRegister delete a Registerer  in the Registers map
func (r *BasicPeerRegistration) delRegister(id peerlib.ID) {
	r.Lock()
	defer r.Unlock()
	_, ok := r.peers[id]
	if ok { // keep LatestRegisterTime for last record purpose
		delete(r.peers, id)
	}
}

// updateRegistersExpiry mark Registered false when time is expired
func (r *BasicPeerRegistration) updateRegistersExpiry() {
	r.Lock()
	defer r.Unlock()
	for id, status := range r.peers {
		if nil != status && status.Registered {
			expire := status.LatestRegisterTime.Add(r.expireDuration)
			if time.Since(expire) > 0 { //expire
				status.Registered = false
				util.LogDebug(r.Log, util.CoWhite, fmt.Sprintf("IsExpire ID:%v is expire", id.ShortString()))
			}
		}
	}
}
