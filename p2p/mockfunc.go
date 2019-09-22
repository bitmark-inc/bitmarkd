package p2p

import peerlib "github.com/libp2p/go-libp2p-core/peer"

// TODO: for rpc node.go
// GetCounts - return connection counts:
//   incoming - total peers connectng to all listeners
//   outgoing - total outgoing connections
func GetCounts() (uint64, uint64) {
	return 0, 0
}

// TODO : for rpc httphandler.go
// BlockHeight - return global block height
func BlockHeight() uint64 {
	return 0
}

//ID return this node host ID
func ID() peerlib.ID {
	return globalData.Host.ID()
}

// FetchConnectors - obtain a list of all connector clients
func FetchConnectors() []peerlib.AddrInfo {
	return globalData.peerInfos()
}
