// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"

	"github.com/bitmark-inc/bitmarkd/blockrecord"
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
func Subscribe(
	i int,
	connectTo string,
	v6 bool,
	serverPublicKey []byte,
	publicKey []byte,
	privateKey []byte,
	log *logger.L,
	proofer Proofer,
) error {

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

	// basic socket options
	socket.SetIpv6(v6)

	// keep-alive settings
	socket.SetTcpKeepalive(1)
	socket.SetTcpKeepaliveCnt(5)
	socket.SetTcpKeepaliveIdle(60)
	socket.SetTcpKeepaliveIntvl(60)

	// ***** FIX THIS: enabling this causes complete failure
	// ***** FIX THIS: socket disconnects, perhaps after IVL value
	// heartbeat
	// socket.SetHeartbeatIvl(heartbeatInterval)
	// socket.SetHeartbeatTimeout(heartbeatTimeout)
	// socket.SetHeartbeatTtl(heartbeatTTL)

	// set subscription prefix - empty => receive everything
	socket.SetSubscribe("")

	socket.Connect(connectTo)
	if nil != err {
		socket.Close()
	}

	// to submit hashing requests
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
			logger.PanicIfError("subscriber", err)
			log.Infof("received data: %s", data)

			// ***** FIX THIS: just debugging? or really split block into multiple nonce ranges
			var item PublishedItem
			err = json.Unmarshal([]byte(data), &item)
			log.Infof("received : %v", item)

			// prevent queuing outdated request
			if !proofer.isWorking() {
				log.Infof("Rest time, discard request")
				continue
			}

			// initial try just forward block
			_, err = proof.Send(mySubmitterIdentity, zmq.SNDMORE)
			logger.PanicIfError("subscriber sending 1", err)
			_, err = proof.Send(data, 0)
			logger.PanicIfError("subscriber sending 2", err)
			ProofQueueIncrement()
			log.Infof("queue depth: %d", proofQueueDepth)

		}
	}()
	return nil
}
