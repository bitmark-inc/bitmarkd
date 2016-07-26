// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/logger"
	zmq "github.com/pebbe/zmq4"
)

// sent by bitmarkd
// ***** FIX THIS: need to refactor
type PublishedItem struct {
	Job    string
	Header blockrecord.Header
}

// subscriber thread
func Subscribe(i int, connectTo string, v6 bool, serverPublicKey []byte, publicKey []byte, privateKey []byte, log *logger.L) error {

	log.Info("starting…")

	socket, err := zmq.NewSocket(zmq.SUB)
	if nil != err {
		return err
	}

	log.Infof("connect to: %q", connectTo)

	socket.SetCurveServer(0)
	socket.SetCurvePublickey(string(publicKey))
	socket.SetCurveSecretkey(string(privateKey))
	socket.SetCurveServerkey(string(serverPublicKey))

	socket.SetIdentity(string(publicKey)) // just use public key for identity

	// // basic socket options
	// //socket.SetIpv6(true)  // ***** FIX THIS find fix for FreeBSD libzmq4 ****
	// socket.SetSndtimeo(SEND_TIMEOUT)
	// socket.SetLinger(LINGER_TIME)
	// socket.SetRouterMandatory(0)   // discard unroutable packets
	// socket.SetRouterHandover(true) // allow quick reconnect for a given public key
	// socket.SetImmediate(false)     // queue messages sent to disconnected peer

	// set subscription prefix - empty => receive everything
	socket.SetSubscribe("")

	socket.Connect(connectTo)
	if nil != err {
		socket.Close()
	}

	// to submitt hashing requests
	proof, err := zmq.NewSocket(zmq.PUSH)
	if nil != err {
		socket.Close()
		return err
	}

	identity := fmt.Sprintf("subscriber-%d", i)
	mySubmitterIdentity := fmt.Sprintf("submitter-%d", i) // ***** FIX THIS: sync up with submitter so names match *****

	proof.SetLinger(0)
	proof.SetIdentity(identity)
	err = proof.Connect(proofRequest)
	if nil != err {
		socket.Close()
		proof.Close()
	}

	// background process
	go func() {
		defer socket.Close()
		defer proof.Close()

		for {
			data, err := socket.Recv(0)
			fault.PanicIfError("subscriber", err)
			log.Infof("received data: %s", data)

			// ***** FIX THIS: just debugging? or really split block into multiple nonce ranges
			var item PublishedItem
			err = json.Unmarshal([]byte(data), &item)
			log.Infof("received : %v", item)

			// initial try just forward block
			_, err = proof.Send(mySubmitterIdentity, zmq.SNDMORE)
			fault.PanicIfError("subscriber sending 1", err)
			_, err = proof.Send(data, 0)
			fault.PanicIfError("subscriber sending 2", err)
			ProofQueueIncrement()
			log.Infof("queue depth: %d", proofQueueDepth)
		}
	}()
	return nil
}
