// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"

	zmq "github.com/pebbe/zmq4"

	"github.com/bitmark-inc/logger"
)

const (
	submission = "inproc://proof.submit" // to fair-queue found proof submissions
	subdeal    = "inproc://proof.dealer" // to route to specific submitter
)

// routes messages to the correct Submitter
func SubmitQueue() {
	go func() {
		err := submitForwarder()
		logger.PanicIfError("proofProxy", err)
	}()
}

// internal submit forwarding loop
func submitForwarder() error {
	in, err := zmq.NewSocket(zmq.PULL)
	if nil != err {
		return err
	}
	defer in.Close()

	in.SetLinger(0)
	err = in.Bind(submission)
	if nil != err {
		return err
	}

	// route messages to correct submitter
	// so packet out of pull _MUST_ have id frame as first item
	// other end is DEALER
	out, err := zmq.NewSocket(zmq.ROUTER)
	if nil != err {
		return err
	}
	defer out.Close()

	out.SetLinger(0)
	err = out.Bind(subdeal)
	if nil != err {
		return err
	}

	// possibly use this: ProxySteerable(frontend, backend, capture, control *Socket) error
	// with a control socket for clean shutdown
	return zmq.Proxy(in, out, nil)
}

// submitter thread
func Submitter(i int, connectTo string, v6 bool, serverPublicKey []byte, publicKey []byte, privateKey []byte, log *logger.L) error {

	log.Info("starting…")

	// socket to dequeue submissions
	dequeue, err := zmq.NewSocket(zmq.DEALER)
	if nil != err {
		return err
	}

	identity := fmt.Sprintf("submitter-%d", i)
	dequeue.SetLinger(0)
	dequeue.SetIdentity(identity) // set the identity of this thread

	err = dequeue.Connect(subdeal)
	if nil != err {
		dequeue.Close()
		return err
	}

	log.Infof("connect to: %q", connectTo)

	rpc, err := zmq.NewSocket(zmq.REQ)
	if nil != err {
		dequeue.Close()
		return err
	}

	// set encryption
	rpc.SetCurveServer(0)
	rpc.SetCurvePublickey(string(publicKey))
	rpc.SetCurveSecretkey(string(privateKey))
	rpc.SetCurveServerkey(string(serverPublicKey))
	log.Infof("*client public:  %x", publicKey)
	log.Tracef("*client private: %x", privateKey)
	log.Infof("*server public:  %x", serverPublicKey)

	// just use public key for identity
	rpc.SetIdentity(string(publicKey))

	// basic socket options
	rpc.SetIpv6(v6)

	// keep-alive settings
	rpc.SetTcpKeepalive(1)
	rpc.SetTcpKeepaliveCnt(5)
	rpc.SetTcpKeepaliveIdle(60)
	rpc.SetTcpKeepaliveIntvl(60)

	rpc.Connect(connectTo)
	if nil != err {
		dequeue.Close()
		rpc.Close()
		return err
	}

	// background process
	go func() {
		defer dequeue.Close()
		defer rpc.Close()

	dequeue_items:
		for {
			request, err := dequeue.RecvMessageBytes(0)
			logger.PanicIfError("dequeue.RecvMessageBytes", err)
			log.Debugf("received data: %x", request)

			// safety check
			if identity != string(request[0]) {
				log.Errorf("received data for wrong submitter: %q  expected: %q", request[0], identity)
				continue dequeue_items
			}

			// compose a request for bitmarkd
			toSend := struct {
				Request string
				Job     string
				Packed  []byte
			}{
				Request: "block.nonce",
				Job:     string(request[1]),
				Packed:  request[2],
			}

			data, err := json.Marshal(toSend)
			if nil != err {

				log.Errorf("JSON encode error: %s", err)
				continue dequeue_items
			}
			log.Infof("rpc: json to send: %s", data)

			_, err = rpc.SendBytes(data, 0)
			logger.PanicIfError("rpc send", err)

			// server response
			response, err := rpc.Recv(0)
			logger.PanicIfError("rpc recv", err)
			log.Debugf("rpc: received data: %s", response)

			var r interface{}
			err = json.Unmarshal([]byte(response), &r)
			logger.PanicIfError("unmarshal response: error: ", err)
			log.Infof("rpc: received from server: %v", r)
		}

	}()
	return nil
}

func SubmitterP2P(i int, v6 bool, log *logger.L, resultCh chan<- []byte) error {
	log.Info("starting…")

	// socket to dequeue submissions
	dequeue, err := zmq.NewSocket(zmq.DEALER)
	if nil != err {
		return err
	}

	identity := fmt.Sprintf("submitter-%d", i)
	dequeue.SetLinger(0)
	dequeue.SetIdentity(identity) // set the identity of this thread

	err = dequeue.Connect(subdeal)
	if nil != err {
		dequeue.Close()
		return err
	}

	// background process
	go func() {
		defer dequeue.Close()

	dequeue_items:
		for {
			request, err := dequeue.RecvMessageBytes(0)
			logger.PanicIfError("dequeue.RecvMessageBytes", err)
			log.Debugf("received data: %x", request)

			// safety check
			if identity != string(request[0]) {
				log.Errorf("received data for wrong submitter: %q  expected: %q", request[0], identity)
				continue dequeue_items
			}

			// compose a request for bitmarkd
			toSend := struct {
				Request string
				Job     string
				Packed  []byte
			}{
				Request: "block.nonce",
				Job:     string(request[1]),
				Packed:  request[2],
			}

			data, err := json.Marshal(toSend)
			if nil != err {
				log.Errorf("JSON encode error: %s", err)
				continue dequeue_items
			}
			log.Infof("rpc: json to send: %s", data)

			resultCh <- data

			// server response
			//response, err := rpc.Recv(0)
			//logger.PanicIfError("rpc recv", err)
			//log.Debugf("rpc: received data: %s", response)
			//
			//var r interface{}
			//err = json.Unmarshal([]byte(response), &r)
			//logger.PanicIfError("unmarshal response: error: ", err)
			//log.Infof("rpc: received from server: %v", r)
		}

	}()
	return nil
}
