// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package peer

import (
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/peer/upstream"
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

func connectToUpstream(log *logger.L, clients []*upstream.Upstream, dynamicStart int, priority string, serverPublicKey []byte, addresses []byte) error {

	log.Infof("connect: %s to: %x @ %x", priority, serverPublicKey, addresses)

	// extract the first valid address
	var address *util.Connection

extract_addresses:
	for {
		conn, n := util.PackedConnection(addresses).Unpack()
		addresses = addresses[n:]

		// ***** FIX THIS: could select for IPv4 or IPv6 here
		// ***** FIX THIS: need to get preference e.g. if have IPv6 the prefer IPv6
		if nil != conn {
			address = conn
			break extract_addresses
		}
		if n <= 0 {
			break
		}
		log.Errorf("reconnect: %x (conn: %x)  error: address is nil", serverPublicKey, conn)
	}

	if nil == address {
		log.Errorf("reconnect: %s  error: no addresses found", serverPublicKey)
		return fault.ErrAddressIsNil
	}

	// if entry matches a static entry, then ignore with warning log
	if dynamicStart > 0 {
		for _, client := range clients[:dynamicStart] {
			if client.IsConnectedTo(serverPublicKey) {
				log.Warnf("ignore change to: %x @ %s", serverPublicKey, *address)
				return nil // no more action is needed
			}
		}
	}

	// detect priority
	offset := toOffset(priority)

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
	log.Infof("reconnect: %x @ %s", serverPublicKey, *address)
	err := dynamic[offset].Connect(address, serverPublicKey)
	if nil != err {
		log.Errorf("ConnectTo: %x @ %s  error: %s", serverPublicKey, *address, err)
	}
	return err
}

func connectToPublisher(log *logger.L, chain string, clients []*zmqutil.Client, dynamicStart int, priority string, serverPublicKey []byte, addresses []byte) error {

	log.Infof("connect: %s to: %x @ %x", priority, serverPublicKey, addresses)

	// extract the first valid address
	var address *util.Connection

extract_addresses:
	for {
		conn, n := util.PackedConnection(addresses).Unpack()
		addresses = addresses[n:]

		// ***** FIX THIS: could select for IPv4 or IPv6 here
		// ***** FIX THIS: need to get preference e.g. if have IPv6 the prefer IPv6
		if nil != conn {
			address = conn
			break extract_addresses
		}
		if n <= 0 {
			break
		}
		log.Errorf("reconnect: %x (conn: %x)  error: address is nil", serverPublicKey, conn)
	}

	if nil == address {
		log.Errorf("reconnect: %s  error: no addresses found", serverPublicKey)
		return fault.ErrAddressIsNil
	}

	// if entry matches a static entry, then ignore with warning log
	if dynamicStart > 0 {
		for _, client := range clients[:dynamicStart] {
			if client.IsConnectedTo(serverPublicKey) {
				log.Warnf("ignore change to: %x @ %s", serverPublicKey, *address)
				return nil // no more action is needed
			}
		}
	}

	// detect priority
	offset := toOffset(priority)

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
	log.Infof("reconnect: %x @ %s", serverPublicKey, *address)
	err := dynamic[offset].Connect(address, serverPublicKey, chain)
	if nil != err {
		log.Errorf("ConnectTo: %x @ %s  error: %s", serverPublicKey, *address, err)
	}
	return err
}

func toOffset(priority string) int {
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
		logger.Panicf("invalid priority: %s", priority)
	}
	return offset
}
