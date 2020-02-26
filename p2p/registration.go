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
	isExpire(id peerlib.ID) bool
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
	peers := []peerlib.ID{}
	for id, status := range r.peers {
		if status.Registered {
			peers = append(peers, id)
		}
	}
	return peers
}

// IsRegistered return if the given ID peer registered
func (r *BasicPeerRegistration) IsRegistered(id peerlib.ID) (registered bool) {
	r.Lock()
	if status, ok := r.peers[id]; ok && status != nil && status.Registered {
		registered = true
	}
	r.Unlock()
	return
}

func (r *BasicPeerRegistration) addRegister(id peerlib.ID) {
	r.Lock()
	r.peers[id] = &RegistrationStatus{Registered: true, LatestRegisterTime: time.Now()}
	r.Unlock()
	//util.LogInfo(r.Log, util.CoCyan, fmt.Sprintf("addRegister ID:%s Registered:%v time:%v", id.ShortString(), r.peers[id].Registered, r.peers[id].LatestRegisterTime.String()))
}

//unRegister unRegister change a peers's  Registered status  to false,  but it doe not not delete the register in the Registers
func (r *BasicPeerRegistration) unRegister(id peerlib.ID) {
	r.Lock()
	status, ok := r.peers[id]
	if ok && status != nil { // keep LatestRegisterTime for last record purpose
		status.Registered = false
	}
	r.Unlock()
}

//delRegister delete a Registerer  in the Registers map
func (r *BasicPeerRegistration) delRegister(id peerlib.ID) {
	r.Lock()
	_, ok := r.peers[id]
	if ok { // keep LatestRegisterTime for last record purpose
		delete(r.peers, id)
	}
	r.Unlock()
}

//isExpire is the register expire
func (r *BasicPeerRegistration) isExpire(id peerlib.ID) bool {
	if status, ok := r.peers[id]; ok && status != nil && status.Registered {
		expire := status.LatestRegisterTime.Add(r.expireDuration)
		passInterval := time.Since(expire)
		if passInterval > 0 { // expire
			return true
		}
	}
	return false
}

//updateRegistersExpiry mark Registered false when time is expired
func (r *BasicPeerRegistration) updateRegistersExpiry() {
	for id, status := range r.peers {
		if r.isExpire(id) { //Keep time for record of last registered time
			r.Lock()
			status.Registered = false
			r.Unlock()
			util.LogDebug(r.Log, util.CoWhite, fmt.Sprintf("IsExpire ID:%v is expire", id.ShortString()))
		}
	}
}
