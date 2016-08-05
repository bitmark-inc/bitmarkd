// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package peer

import (
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/logger"
)

// priority offsets
const (
	offsetNeighbour1 = iota
	offsetNeighbour3 = iota
	offsetCross25    = iota
	offsetCross50    = iota
	offsetCross75    = iota
	//this is the number of items, add new offsets above this line
	offsetCount = iota
)

// reconnect clients
// or make a new connection if not already connected to that node
// will recycle the oldest dynamic connection if no free nodes
// priority is an offset from dynamic start
func connectTo(log *logger.L, clients []*zmqutil.Client, dynamicStart int, priority string, serverPublicKey []byte, addresses []byte) {

	log.Infof("***** connect: %s to: %x @ %x", priority, serverPublicKey, addresses)

	const maximumConnections = 5

	connect := make([]*util.Connection, maximumConnections)

	for i := 0; i < len(connect); i += 1 {
		conn, n := util.PackedConnection(addresses).Unpack()
		addresses = addresses[n:]
		connect[i] = conn
	}

	// ***** FIX THIS: connection logic
	// maybe should pick a random one of the connections
	// maybe check if IPv4 or IPv6 preferred

	address := connect[0]
	//free := (*zmqutil.Client)(nil)
	//oldest := (*zmqutil.Client)(nil)

	// if entry matches a static entry, then ignore with warning log
	if dynamicStart > 0 {
		for _, client := range clients[:dynamicStart] {
			if client.IsConnectedTo(serverPublicKey) {
				log.Warnf("ignore change to: %x @ %s", serverPublicKey, *address)
				return // no more action is needed
			}
		}
	}

	// detect priority
	offset := offsetNeighbour1
	switch priority {
	case "N1":
		offset = offsetNeighbour1
	case "N3":
		offset = offsetNeighbour3
	case "X25":
		offset = offsetCross25
	case "X50":
		offset = offsetCross50
	case "X75":
		offset = offsetCross75
	default:
		log.Criticalf("invalid priority: %s", priority)
		fault.Panicf("invalid priority: %s", priority)
	}

	// scan dynamic clients to see if already connected
	actual := offset
	dynamic := clients[dynamicStart:]
dynamicScan:
	for i, client := range dynamic {
		if client.IsConnectedTo(serverPublicKey) {
			actual = i
			break dynamicScan
		}
	}

	// swap clients if necessary
	log.Infof("offset: %d  actual: %d", offset, actual)
	if actual != offset {
		dynamic[actual], dynamic[offset] = dynamic[offset], dynamic[actual]
	}

	// reconnect the one corresponding to priority
	log.Infof("***** reconnect: %x @ %s", serverPublicKey, *address)
	err := dynamic[offset].Connect(address, serverPublicKey)
	if nil != err {
		log.Errorf("ConnectTo: %x @ %s  error: %v", serverPublicKey, *address, err)
	}
}
