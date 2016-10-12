// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package peer

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/genesis"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/logger"
	zmq "github.com/pebbe/zmq4"
	"time"
)

// various timeouts
const (
	cycleInterval       = 10 * time.Second       // pause to limit bandwidth
	connectorTimeout    = 500 * time.Millisecond // time out for connections
	samplelingLimit     = 10                     // number of cycles to be 1 block out of sync before resync
	fetchBlocksPerCycle = 50                     // number of blocks to fetch in one set
)

// a state type for the thread
type connectorState int

// state of the connector process
const (
	cStateConnecting   connectorState = iota // register to nodes and make outgoing connections
	cStateHighestBlock connectorState = iota // locate node(s) with highest block number
	cStateForkDetect   connectorState = iota // read block hashes to check for possible fork
	cStateFetchBlocks  connectorState = iota // fetch blocks from current or fork point
	cStateRebuild      connectorState = iota // rebuild database from fork point (config setting to force total rebuild)
	cStateSampling     connectorState = iota // signal resync complete and sample nodes to see if out of sync occurs
)

type connector struct {
	log          *logger.L
	clients      []*zmqutil.Client
	dynamicStart int
	state        connectorState

	theClient          *zmqutil.Client // client to fetch blocak data from
	startBlockNumber   uint64          // block number wher local chain forks
	highestBlockNumber uint64          // block number on best node
	samples            int             // counter to detect missed block broadcast
}

// initialise the connector
func (conn *connector) initialise(privateKey []byte, publicKey []byte, connect []Connection, dynamicEnabled bool) error {

	log := logger.New("connector")
	if nil == log {
		return fault.ErrInvalidLoggerChannel
	}
	conn.log = log

	log.Info("initialising…")

	// allocate all sockets
	staticCount := len(connect) // can be zero
	if 0 == staticCount && !dynamicEnabled {
		log.Error("zero static connections and dynamic is disabled")
		return fault.ErrNoConnectionsAvailable
	}
	conn.clients = make([]*zmqutil.Client, staticCount+offsetCount)
	conn.dynamicStart = staticCount // index of first dynamic socket

	// error code for goto fail
	errX := error(nil)

	// initially connect all static sockets
	for i, c := range connect {
		address, err := util.NewConnection(c.Address)
		if nil != err {
			log.Errorf("client[%d]=address: %q  error: %v", i, c.Address, err)
			errX = err
			goto fail
		}
		serverPublicKey, err := hex.DecodeString(c.PublicKey)
		if nil != err {
			log.Errorf("client[%d]=public: %q  error: %v", i, c.PublicKey, err)
			errX = err
			goto fail
		}

		// prevent connection to self
		if bytes.Equal(publicKey, serverPublicKey) {
			errX = fault.ErrConnectingToSelfForbidden
			log.Errorf("client[%d]=public: %q  error: %v", i, c.PublicKey, errX)
			goto fail
		}

		client, err := zmqutil.NewClient(zmq.REQ, privateKey, publicKey, connectorTimeout)
		if nil != err {
			log.Errorf("client[%d]=%q  error: %v", i, address, err)
			errX = err
			goto fail
		}

		conn.clients[i] = client

		err = client.Connect(address, serverPublicKey)
		if nil != err {
			log.Errorf("connect[%d]=%q  error: %v", i, address, err)
			errX = err
			goto fail
		}
		log.Infof("public key: %x  at: %q", serverPublicKey, c.Address)
	}

	// just create sockets for dynamic clients
	for i := conn.dynamicStart; i < len(conn.clients); i += 1 {
		client, err := zmqutil.NewClient(zmq.REQ, privateKey, publicKey, connectorTimeout)
		if nil != err {
			log.Errorf("client[%d]  error: %v", i, err)
			errX = err
			goto fail
		}

		conn.clients[i] = client
	}

	// start state machine
	conn.state = cStateConnecting

	return nil

	// error handling
fail:
	zmqutil.CloseClients(conn.clients)
	return errX
}

// various RPC calls to upstream connections
func (conn *connector) Run(args interface{}, shutdown <-chan struct{}) {

	log := conn.log

	log.Info("starting…")

	queue := messagebus.Bus.Connector.Chan()

loop:
	for {
		// wait for shutdown
		log.Info("waiting…")

		select {
		case <-shutdown:
			break loop
		case item := <-queue:
			conn.log.Infof("received: %s  public key: %x  connect: %x", item.Command, item.Parameters[0], item.Parameters[1])
			connectTo(conn.log, conn.clients, conn.dynamicStart, item.Command, item.Parameters[0], item.Parameters[1])

		case <-time.After(cycleInterval):
			conn.process()
		}
	}
	zmqutil.CloseClients(conn.clients)
}

// process the connect and return response
func (conn *connector) process() {
	log := conn.log

	log.Infof("current state: %s", conn.state)

	switch conn.state {
	case cStateConnecting:
		mode.Set(mode.Resynchronise)
		if register(log, conn.clients) {
			conn.state += 1
		}
	case cStateHighestBlock:
		conn.highestBlockNumber, conn.theClient = highestBlock(log, conn.clients)
		if conn.highestBlockNumber > 0 && nil != conn.theClient {
			conn.state += 1
		}
		log.Infof("highest block number: %d", conn.highestBlockNumber)

	case cStateForkDetect:
		h := block.GetHeight()
		if conn.highestBlockNumber < h {
			conn.state = cStateRebuild
		} else {
			// first block number
			conn.startBlockNumber = genesis.BlockNumber + 1
			conn.state += 1 // assume success
			log.Infof("block number: %d", h)

			// check digests of descending blocks (to detect a fork)
			for ; h > genesis.BlockNumber; h -= 1 {
				digest, err := block.DigestForBlock(h)
				if nil != err {
					log.Infof("block number: %d  local digest error: %v", h, err)
					conn.state = cStateHighestBlock // retry
					break
				}
				d, err := blockDigest(conn.theClient, h)
				if nil != err {
					log.Infof("block number: %d  fetch digest error: %v", h, err)
					conn.state = cStateHighestBlock // retry
					break
				} else if d == digest {
					conn.startBlockNumber = h + 1
					log.Infof("fork from block number: %d", conn.startBlockNumber)

					// remove old blocks
					err := block.DeleteDownToBlock(conn.startBlockNumber)
					if nil != err {
						log.Errorf("delete down to block number: %d  error: %v", conn.startBlockNumber, err)
						conn.state = cStateHighestBlock // retry
					}
					break
				}

			}
		}

	case cStateFetchBlocks:
		for n := 0; n < fetchBlocksPerCycle; n += 1 {

			if conn.startBlockNumber > conn.highestBlockNumber {
				conn.state += 1 // assume success
				break
			}

			log.Infof("fetch block number: %d", conn.startBlockNumber)
			packedBlock, err := blockData(conn.theClient, conn.startBlockNumber)
			if nil != err {
				log.Errorf("fetch block number: %d  error: %v", conn.startBlockNumber, err)
				conn.state = cStateHighestBlock // retry
				break
			}
			log.Debugf("store block number: %d", conn.startBlockNumber)
			err = block.StoreIncoming(packedBlock)
			if nil != err {
				log.Errorf("store block number: %d  error: %v", conn.startBlockNumber, err)
				conn.state = cStateHighestBlock // retry
				break
			}

			// next block
			conn.startBlockNumber += 1

		}

	case cStateRebuild:
		// return to normal operations
		conn.state += 1  // next state
		conn.samples = 0 // zero out the counter
		mode.Set(mode.Normal)

	case cStateSampling:
		// check peers
		conn.highestBlockNumber, conn.theClient = highestBlock(log, conn.clients)
		height := block.GetHeight()

		log.Infof("remote height: %d", conn.highestBlockNumber)
		log.Infof("local height: %d", height)

		if conn.highestBlockNumber > height {
			if conn.highestBlockNumber-height >= 2 {
				conn.state = cStateForkDetect
			} else {
				conn.samples += 1
				if conn.samples > samplelingLimit {
					conn.state = cStateForkDetect
				}
			}
		}

	}
	log.Infof("next state: %s", conn.state)
}

// ***** FIX THIS: is this needed
// func CheckServer(client *zmqutil.Client) error {

// 	err := client.Send("I")
// 	if nil != err {
// 		return err
// 	}
// 	data, err := client.Receive(0)
// 	if nil != err {
// 		return err
// 	}

// 	switch string(data[0]) {
// 	case "E":
// 		return fault.InvalidError(string(data[1]))
// 	case "I":
// 		var info serverInfo
// 		err = json.Unmarshal(data[1], &info)
// 		if nil != err {
// 			return err
// 		}

// 		if info.Chain != mode.ChainName() {
// 			return fault.ErrIncorrectChain
// 		}
// 		return nil
// 	default:
// 	}
// 	return fault.ErrInvalidPeerResponse
// }

// send a registration request to all connected clients
func register(log *logger.L, clients []*zmqutil.Client) bool {
	n := 0
	for i, client := range clients {
		log.Infof("register trying client: %d", i)
		if !client.IsConnected() {
			log.Info("not connected")
			continue
		}

		err := announce.SendRegistration(client, "R")
		if nil != err {
			log.Errorf("send registration error: %v", err)
			client.Reconnect()
			continue
		}
		data, err := client.Receive(0)
		if nil != err {
			log.Errorf("send registration receive error: %v", err)
			client.Reconnect()
			continue
		}
		switch string(data[0]) {
		case "E":
			if 2 == len(data) {
				log.Errorf("register error: %q", data[1])
			}
			continue
		case "R":
			if 5 != len(data) {
				log.Errorf("register response incorrect: %x", data)
				continue
			}
			n += 1
			chain := mode.ChainName()
			received := string(data[1])
			if received != chain {
				log.Criticalf("expected chain: %q but received: %q", chain, received)
				fault.Panicf("expected chain: %q but received: %q", chain, received)
			}
			log.Infof("register replied: %x:  broadcasts: %x  listeners: %x", data[2], data[3], data[4])
			announce.AddPeer(data[2], data[3], data[4]) // publicKey, broadcasts, listeners
		default:
			continue
		}
	}
	return n > 0 // if registration occured
}

// determine client with highest block
func highestBlock(log *logger.L, clients []*zmqutil.Client) (uint64, *zmqutil.Client) {

	h := uint64(0)
	c := (*zmqutil.Client)(nil)

	for _, client := range clients {
		if !client.IsConnected() {
			continue
		}

		err := client.Send("N")
		if nil != err {
			log.Errorf("highestBlock: send error: %v", err)
			client.Reconnect()
			continue
		}

		data, err := client.Receive(0)
		if nil != err {
			log.Errorf("highestBlock: receive error: %v", err)
			client.Reconnect()
			continue
		}
		if 2 != len(data) {
			log.Errorf("highestBlock: received: %d  expected: 2", len(data))
			continue
		}
		switch string(data[0]) {
		case "E":
			log.Errorf("highestBlock: rpc error response: %q", data[1])
			continue
		case "N":
			if 8 != len(data[1]) {
				continue
			}
			n := binary.BigEndian.Uint64(data[1])

			if n > h {
				h = n
				c = client
			}
		default:
		}
	}
	return h, c
}

// fetch block digest
func blockDigest(client *zmqutil.Client, blockNumber uint64) (blockdigest.Digest, error) {
	parameter := make([]byte, 8)
	binary.BigEndian.PutUint64(parameter, blockNumber)
	err := client.Send("H", parameter)
	if nil != err {
		client.Reconnect()
		return blockdigest.Digest{}, err
	}

	data, err := client.Receive(0)
	if nil != err {
		client.Reconnect()
		return blockdigest.Digest{}, err
	}

	if 2 != len(data) {
		return blockdigest.Digest{}, fault.ErrInvalidPeerResponse
	}

	switch string(data[0]) {
	case "E":
		return blockdigest.Digest{}, fault.InvalidError(string(data[1]))
	case "H":
		d := blockdigest.Digest{}
		if blockdigest.Length == len(data[1]) {
			err := blockdigest.DigestFromBytes(&d, data[1])
			return d, err
		}
	default:
	}
	return blockdigest.Digest{}, fault.ErrInvalidPeerResponse
}

// fetch block data
func blockData(client *zmqutil.Client, blockNumber uint64) ([]byte, error) {
	parameter := make([]byte, 8)
	binary.BigEndian.PutUint64(parameter, blockNumber)
	err := client.Send("B", parameter)
	if nil != err {
		client.Reconnect()
		return nil, err
	}

	data, err := client.Receive(0)
	if nil != err {
		client.Reconnect()
		return nil, err
	}

	if 2 != len(data) {
		return nil, fault.ErrInvalidPeerResponse
	}

	switch string(data[0]) {
	case "E":
		return nil, fault.InvalidError(string(data[1]))
	case "B":
		return data[1], nil
	default:
	}
	return nil, fault.ErrInvalidPeerResponse
}

func (state connectorState) String() string {
	switch state {
	case cStateConnecting:
		return "Connecting"
	case cStateHighestBlock:
		return "HighestBlock"
	case cStateForkDetect:
		return "ForkDetect"
	case cStateFetchBlocks:
		return "FetchBlocks"
	case cStateRebuild:
		return "Rebuild"
	case cStateSampling:
		return "Sampling"
	default:
		return "*Unknown*"
	}
}
