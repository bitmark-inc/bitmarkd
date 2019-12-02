package p2p

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/bitmark-inc/bitmarkd/counter"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/logger"
	p2pcore "github.com/libp2p/go-libp2p-core"
	p2pnet "github.com/libp2p/go-libp2p-core/network"
	peerlib "github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

// MetricsNetwork contain P2P network metrics
type MetricsNetwork struct {
	mutex         *sync.Mutex
	Log           *logger.L
	streamCount   counter.Counter
	connCount     counter.Counter
	connectStatus map[peerlib.ID]bool
}

//NewMetricsNetwork  create a new MetricsNetwork and start to monitor the connection and stream
func NewMetricsNetwork(host p2pcore.Host, log *logger.L) *MetricsNetwork {
	var mutex = &sync.Mutex{}
	metrics := MetricsNetwork{mutex: mutex, Log: log, connectStatus: make(map[peerlib.ID]bool)}
	go metrics.monitor(host, log)
	return &metrics
}

func (m *MetricsNetwork) monitor(host p2pcore.Host, log *logger.L) {
	host.Network().Notify(&p2pnet.NotifyBundle{
		ListenF: func(net p2pnet.Network, addr ma.Multiaddr) {
			util.LogDebug(log, util.CoReset, fmt.Sprintf("@@Host: %v is listen at %v\n", addr.String(), time.Now()))
		},
		ConnectedF: func(net p2pnet.Network, conn p2pnet.Conn) {
			m.connCount.Increment()
			globalData.setConnectStatus(conn.RemotePeer(), true)
			util.LogInfo(log, util.CoReset, fmt.Sprintf("@@: Conn: ID:%v Addr %v CONNECTED at %v ConnCount:%d\n", conn.RemotePeer().ShortString(), conn.RemoteMultiaddr().String(), time.Now(), m.connCount))
		},
		DisconnectedF: func(net p2pnet.Network, conn p2pnet.Conn) {
			m.connCount.Decrement()
			globalData.setConnectStatus(conn.RemotePeer(), false)
			util.LogInfo(log, util.CoReset, fmt.Sprintf("@@: Conn: ID:%v Addr %v DISCONNECTED at %v ConnCount:%d\n", conn.RemotePeer().ShortString(), conn.RemoteMultiaddr().String(), time.Now(), m.connCount))
		},
		OpenedStreamF: func(net p2pnet.Network, stream p2pnet.Stream) {
			m.streamCount.Increment()
			//util.LogDebug(log, util.CoReset, fmt.Sprintf("@@Stream : %v-%v is Opened at %v streamCount:%d\n", stream.Conn().RemoteMultiaddr().String(), stream.Protocol(), time.Now(), m.streamCount))
		},
		ClosedStreamF: func(net p2pnet.Network, stream p2pnet.Stream) {
			m.streamCount.Decrement()
			//util.LogDebug(log, util.CoReset, fmt.Sprintf("@@Stream :%v-%v is Closed at %v streamCount:%d\n", stream.Conn().RemoteMultiaddr().String(), stream.Protocol(), time.Now(), m.streamCount))
		},
	})
}

func (m *MetricsNetwork) setConnectStatus(id peerlib.ID, status bool) {
	m.mutex.Lock()
	m.connectStatus[id] = status
	m.mutex.Unlock()
}

//IsConnected Return connect status of given peer ID
func (m *MetricsNetwork) IsConnected(id peerlib.ID) bool {
	connected, err := m.ConnectStatus(id)
	if nil == err && connected {
		return true
	}
	return false
}

//ConnectStatus Return connect status of given peer ID
func (m *MetricsNetwork) ConnectStatus(id peerlib.ID) (bool, error) {
	m.mutex.Lock()
	val, ok := m.connectStatus[id]
	m.mutex.Unlock()
	if ok {
		return val, nil
	}
	return false, errors.New("peer ID does not exist")
}

//GetConnCount return current connection counts
func (m *MetricsNetwork) GetConnCount() counter.Counter {
	util.LogInfo(m.Log, util.CoWhite, fmt.Sprintf("@@GetConnCount:%d", m.connCount))
	return m.connCount
}

//GetNetworkMetricConnCount return current connection counts
func GetNetworkMetricConnCount() counter.Counter {
	util.LogInfo(globalData.Log, util.CoWhite, fmt.Sprintf("@@GetConnCount:%d", globalData.MetricsNetwork.connCount))
	return globalData.MetricsNetwork.connCount
}
