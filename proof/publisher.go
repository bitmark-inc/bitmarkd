// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package proof

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/logger"
	zmq "github.com/pebbe/zmq4"
	"time"
)

const (
	publishInterval    = 60 * time.Second
	publisherZapDomain = "publisher"
)

// to send to proofer
// ***** FIX THIS: need to refactor
type PublishedItem struct {
	Job    string
	Header blockrecord.Header
}

type publisher struct {
	log    *logger.L
	socket *zmq.Socket
}

// initialise the publisher
func (pub *publisher) initialise(configuration *Configuration) error {

	log := logger.New("publisher")
	if nil == log {
		return fault.ErrInvalidLoggerChannel
	}
	pub.log = log

	log.Info("initialising…")

	// read the keys
	privateKey, err := zmqutil.ReadKeyFile(configuration.PrivateKey)
	if nil != err {
		return err
	}
	publicKey, err := zmqutil.ReadKeyFile(configuration.PublicKey)
	if nil != err {
		return err
	}

	socket, err := zmq.NewSocket(zmq.PUB)
	if nil != err {
		return err
	}
	pub.socket = socket

	// ***** FIX THIS ****
	// this allows any client to connect
	zmq.AuthAllow(publisherZapDomain, "127.0.0.1/8")
	zmq.AuthCurveAdd(publisherZapDomain, zmq.CURVE_ALLOW_ANY)

	// domain is servers public key
	socket.SetCurveServer(1)
	//socket.SetCurvePublickey(publicKey)
	socket.SetCurveSecretkey(privateKey)
	log.Infof("server public:  %q", publicKey)
	log.Infof("server private: %q", privateKey)

	socket.SetZapDomain(publisherZapDomain)

	socket.SetIdentity(publicKey) // just use public key for identity

	// ***** FIX THIS ****
	// maybe need to change above line to specific keys later
	//   e.g. zmq.AuthCurveAdd(serverPublicKey, client1PublicKey)
	//        zmq.AuthCurveAdd(serverPublicKey, client2PublicKey)
	// perhaps as part of ConnectTo

	// // basic socket options
	// socket.SetIpv6(true) // ***** FIX THIS find fix for FreeBSD libzmq4 ****
	// socket.SetSndtimeo(SEND_TIMEOUT)
	// socket.SetLinger(LINGER_TIME)
	// socket.SetRouterMandatory(0)   // discard unroutable packets
	// socket.SetRouterHandover(true) // allow quick reconnect for a given public key
	// socket.SetImmediate(false)     // queue messages sent to disconnected peer
	for i, address := range configuration.Publish {
		bindTo, err := util.CanonicalIPandPort("tcp://", address)
		if nil != err {
			log.Errorf("publisher[%d]=%q  error: %v", i, address, err)
			return err
		}

		err = socket.Bind(bindTo)
		if nil != err {
			log.Errorf("publish[%d]=%q  error: %v", i, address, err)
			socket.Close()
			return err
		}
		log.Infof("publish on: %q", address)
	}
	return nil
}

// wait for new blocks or new payment items
// to ensure the queue integrity as heap is not thread-safe
func (pub *publisher) Run(args interface{}, shutdown <-chan struct{}) {

	log := pub.log

	log.Info("starting…")

loop:
	for {
		log.Info("waiting…")
		select {
		case <-shutdown:
			break loop
		case <-time.After(publishInterval):
			pub.process()
		}
	}
	pub.socket.Close()
}

// process some items into a block and publish it
func (pub *publisher) process() {

	test := true

	bits := difficulty.New()

	randomBytes := make([]byte, 8)
	_, err := rand.Read(randomBytes)
	nonce := blockrecord.NonceType(binary.LittleEndian.Uint64(randomBytes))

	liveMerkleRoot := merkle.Digest{
		0x63, 0x8c, 0x15, 0x9c, 0x1f, 0x11, 0x3f, 0x70,
		0xa9, 0x86, 0x6d, 0x9a, 0x9e, 0x52, 0xe9, 0xef,
		0xe9, 0xb9, 0x92, 0x08, 0x48, 0xad, 0x1d, 0xf3,
		0x48, 0x51, 0xbe, 0x8a, 0x56, 0x2a, 0x99, 0x8d,
	}
	testMerkleRoot := merkle.Digest{
		0xee, 0x07, 0xbb, 0xc3, 0xd7, 0x49, 0xe0, 0x7d,
		0x24, 0xb9, 0x0c, 0xd1, 0xec, 0x35, 0x14, 0x70,
		0x2e, 0x87, 0x85, 0x22, 0xda, 0xf7, 0x16, 0xc1,
		0x73, 0x24, 0xd6, 0x66, 0x69, 0x7b, 0x8a, 0x63,
	}

	var merkleRoot merkle.Digest
	var timestamp uint64
	job := "unknown"
	if test {
		job = "test"
		merkleRoot = testMerkleRoot
		timestamp = uint64(0x5478424b) // test
		test = false
	} else {
		job = "live"
		merkleRoot = liveMerkleRoot
		timestamp = uint64(0x56809ab7) // live
		test = true
	}

	// PreviousBlock is all zero
	message := PublishedItem{
		Job: job,
		Header: blockrecord.Header{
			Version:          1,
			TransactionCount: 1,
			Number:           1,
			MerkleRoot:       merkleRoot,
			Timestamp:        timestamp,
			Difficulty:       bits,
			Nonce:            nonce,
		},
	}

	data, err := json.Marshal(message)
	fault.PanicIfError("JSON encode error: %v", err)

	pub.log.Infof("json to send: %s\n", data)

	_, err = pub.socket.SendBytes(data, 0|zmq.DONTWAIT)
	fault.PanicIfError("publisher", err)
	time.Sleep(10 * time.Second)
}
