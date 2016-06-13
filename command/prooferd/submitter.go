// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/logger"
	zmq "github.com/pebbe/zmq4"
)

const (
	submission = "inproc://proof.submit" // to fair-queue found proof submissions
	subdeal    = "inproc://proof.dealer" // to route to specific submitter
)

// routes messages to the correct Submitter
func SubmitQueue() {
	go func() {
		err := submitForwarder()
		fault.PanicIfError("proofProxy", err)
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
func Submitter(i int, connectTo string, serverPublicKey string, publicKey string, privateKey string, log *logger.L) error {

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
	rpc.SetCurvePublickey(publicKey)
	rpc.SetCurveSecretkey(privateKey)
	rpc.SetCurveServerkey(serverPublicKey)
	log.Infof("*client public:  %q", publicKey)
	log.Infof("*client private: %q", privateKey)
	log.Infof("*server public:  %q", serverPublicKey)

	// just use public key for identity
	rpc.SetIdentity(publicKey)

	// // basic socket options
	// //socket.SetIpv6(true)  // ***** FIX THIS find fix for FreeBSD libzmq4 ****
	// socket.SetSndtimeo(SEND_TIMEOUT)
	// socket.SetLinger(LINGER_TIME)
	// socket.SetRouterMandatory(0)   // discard unroutable packets
	// socket.SetRouterHandover(true) // allow quick reconnect for a given public key
	// socket.SetImmediate(false)     // queue messages sent to disconnected peer

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

		for {
			request, err := dequeue.RecvMessageBytes(0)
			fault.PanicIfError("dequeue.RecvMessageBytes", err)
			//log.Infof("received data: %s", request)

			// safety check
			if identity != string(request[0]) {
				log.Errorf("received data for wrong submitter: %q  expected: %q", request[0], identity)
				continue
			}

			// compose a request for bitmarkd
			toSend := struct {
				Request string
				Job     string
				Packed  []byte
			}{
				Request: "block.header",
				Job:     string(request[1]),
				Packed:  request[2],
			}

			data, err := json.Marshal(toSend)
			if nil != err {
				log.Errorf("JSON encode error: %v", err)
				continue
			}
			log.Infof("rpc: json to send: %s", data)

			_, err = rpc.SendBytes(data, 0)
			fault.PanicIfError("rpc send", err)

			// server response
			response, err := rpc.Recv(0)
			fault.PanicIfError("rpc recv", err)
			//log.Infof("rpc: received data: %s", response)
			var r interface{}
			err = json.Unmarshal([]byte(response), &r)
			log.Infof("rpc: received from server: %v", r)
		}

	}()
	return nil
}
