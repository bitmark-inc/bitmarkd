// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package upstream

import (
	"encoding/binary"
	"fmt"
	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/counter"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/logger"
	zmq "github.com/pebbe/zmq4"
	"sync"
	"time"
)

const (
	cycleInterval = 30 * time.Second
	queueSize     = 10 // 0 => synchronous queue
)

type Upstream struct {
	sync.RWMutex
	log         *logger.L
	client      *zmqutil.Client
	registered  bool
	blockHeight uint64
	shutdown    chan<- struct{}
}

// atomically incremented counter for log names
var upstreamCounter counter.Counter

func New(privateKey []byte, publicKey []byte, timeout time.Duration) (*Upstream, error) {
	client, err := zmqutil.NewClient(zmq.REQ, privateKey, publicKey, timeout)
	if nil != err {
		return nil, err
	}

	n := upstreamCounter.Increment()

	shutdown := make(chan struct{})
	u := &Upstream{
		log:         logger.New(fmt.Sprintf("upstream@%d", n)),
		client:      client,
		registered:  false,
		blockHeight: 0,
		shutdown:    shutdown,
	}
	go upstreamRunner(u, shutdown)
	return u, nil
}

func (u *Upstream) Destroy() {
	close(u.shutdown)
}

// check the current destination
//
// does not mean actually connected, as could be in a timeout and
// reconnect state
func (u *Upstream) IsConnectedTo(serverPublicKey []byte) bool {
	u.Lock()
	result := u.client.IsConnectedTo(serverPublicKey)
	u.Unlock()
	return result
}

// if registered the have avalid connection
func (u *Upstream) IsOK() bool {
	u.Lock()
	result := u.registered
	u.Unlock()
	return result
}

// if registered the have avalid connection
func (u *Upstream) ConnectedTo() *zmqutil.Connected {
	u.Lock()
	result := u.client.ConnectedTo()
	u.Unlock()
	return result
}

// connect (or reconnect) to a specific server
func (u *Upstream) Connect(address *util.Connection, serverPublicKey []byte) error {
	u.log.Infof("connecting to address: %s", address)
	u.log.Infof("connecting to server: %x", serverPublicKey)
	u.Lock()
	err := u.client.Connect(address, serverPublicKey, mode.ChainName())
	u.Unlock()
	return err
}

// fetch height from last polled value
func (u *Upstream) GetHeight() uint64 {
	u.Lock()
	height := u.blockHeight
	u.Unlock()
	return height
}

// fetch block digest
func (u *Upstream) GetBlockDigest(blockNumber uint64) (blockdigest.Digest, error) {
	parameter := make([]byte, 8)
	binary.BigEndian.PutUint64(parameter, blockNumber)

	// critical section - lock out the runner process
	u.Lock()
	var data [][]byte
	err := u.client.Send("H", parameter)
	if nil == err {
		data, err = u.client.Receive(0)
	}
	u.Unlock()

	if nil != err {
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
func (u *Upstream) GetBlockData(blockNumber uint64) ([]byte, error) {
	parameter := make([]byte, 8)
	binary.BigEndian.PutUint64(parameter, blockNumber)

	// critical section - lock out the runner process
	u.Lock()
	var data [][]byte
	err := u.client.Send("B", parameter)
	if nil == err {
		data, err = u.client.Receive(0)
	}
	u.Unlock()

	if nil != err {
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

func upstreamRunner(u *Upstream, shutdown <-chan struct{}) {
	log := u.log

	log.Info("starting…")

	queue := messagebus.Bus.Broadcast.Chan(queueSize)

loop:
	for {
		log.Info("waiting…")

		select {
		case <-shutdown:
			break loop

		case item := <-queue:
			log.Infof("received: %q  %x", item.Command, item.Parameters)
			if u.registered {
				u.Lock()
				err := push(u.client, u.log, &item)
				if nil != err {
					log.Errorf("push: error: %s", err)
					err := u.client.Reconnect()
					if nil != err {
						log.Errorf("push: reconnect error: %s", err)
					}
				}
				u.Unlock()
			}

		case <-time.After(cycleInterval):
			u.Lock()
			if !u.registered {
				err := register(u.client, u.log)
				if nil != err {
					log.Errorf("register: error: %s", err)
					err := u.client.Reconnect()
					if nil != err {
						log.Errorf("register: reconnect error: %s", err)
					}
					u.Unlock()
					continue loop // try again later
				}
				u.registered = true
			}

			h, err := getHeight(u.client, u.log)
			if nil == err {
				u.blockHeight = h
			} else {
				u.registered = false
				log.Errorf("getHeight: error: %s", err)
				err := u.client.Reconnect()
				if nil != err {
					log.Errorf("highestBlock: reconnect error: %s", err)
				}
			}
			u.Unlock()
		}
	}
	log.Info("shutting down…")
	u.client.Close()
	log.Info("stopped")
}

func register(client *zmqutil.Client, log *logger.L) error {

	log.Debugf("register: client: %s", client)

	err := announce.SendRegistration(client, "R")
	if nil != err {
		return err
	}

	data, err := client.Receive(0)
	if nil != err {
		return err
	}

	if len(data) < 2 {
		return fmt.Errorf("register received: %d  expected at least: 2", len(data))
	}

	switch string(data[0]) {
	case "E":
		return fmt.Errorf("register error: %q", data[1])
	case "R":
		if len(data) < 6 {
			return fmt.Errorf("register response incorrect: %x", data)
		}
		chain := mode.ChainName()
		received := string(data[1])
		if received != chain {
			log.Criticalf("expected chain: %q but received: %q", chain, received)
			logger.Panicf("expected chain: %q but received: %q", chain, received)
		}

		timestamp := binary.BigEndian.Uint64(data[5])
		log.Infof("register replied: %x:  broadcasts: %x  listeners: %x  timestamp: %d", data[2], data[3], data[4], timestamp)
		announce.AddPeer(data[2], data[3], data[4], timestamp) // publicKey, broadcasts, listeners
		return nil
	default:
		return fmt.Errorf("rpc unexpected response: %q", data[0])
	}
}

func getHeight(client *zmqutil.Client, log *logger.L) (uint64, error) {

	log.Debugf("getHeight: client: %s", client)

	err := client.Send("N")
	if nil != err {
		return 0, err
	}

	data, err := client.Receive(0)
	if nil != err {
		return 0, err
	}
	if 2 != len(data) {
		return 0, fmt.Errorf("getHeight received: %d  expected: 2", len(data))
	}

	switch string(data[0]) {
	case "E":
		return 0, fmt.Errorf("rpc error response: %q", data[1])
	case "N":
		if 8 != len(data[1]) {
			return 0, fmt.Errorf("highestBlock: rpc invalid response: %q", data[1])
		}
		height := binary.BigEndian.Uint64(data[1])
		log.Infof("height: %d", height)
		return height, nil
	default:
		return 0, fmt.Errorf("rpc unexpected response: %q", data[0])
	}
}

func push(client *zmqutil.Client, log *logger.L, item *messagebus.Message) error {

	log.Debugf("push: client: %s", client)

	err := client.Send(item.Command, item.Parameters)
	if nil != err {
		return err
	}

	data, err := client.Receive(0)
	if nil != err {
		return err
	}
	if 2 != len(data) {
		return fmt.Errorf("push received: %d  expected: 2", len(data))
	}

	switch string(data[0]) {
	case "E":
		return fmt.Errorf("rpc error response: %q", data[1])
	case "A":
		return nil
	default:
		return fmt.Errorf("rpc unexpected response: %q", data[0])
	}
}
