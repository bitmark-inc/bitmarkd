// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package proof

import (
	"encoding/json"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/logger"
	zmq "github.com/pebbe/zmq4"
)

const (
	submissionZapDomain = "submit"
	submissionSignal    = "inproc://bitmark-submission-signal"
)

type submission struct {
	log    *logger.L
	push   *zmq.Socket
	pull   *zmq.Socket
	socket *zmq.Socket
}

// initialise the submission
func (sub *submission) initialise(configuration *Configuration) error {

	log := logger.New("submission")
	if nil == log {
		return fault.ErrInvalidLoggerChannel
	}
	sub.log = log

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

	// send half signalling channel
	push, err := zmq.NewSocket(zmq.PUSH)
	if nil != err {
		return err
	}
	push.SetLinger(0)
	err = push.Bind(submissionSignal)
	if nil != err {
		return err
	}

	sub.push = push

	// receive half of signalling channel
	pull, err := zmq.NewSocket(zmq.PULL)
	if nil != err {
		return err
	}
	pull.SetLinger(0)
	err = pull.Connect(submissionSignal)
	if nil != err {
		return err
	}

	sub.pull = pull

	socket, err := zmq.NewSocket(zmq.REP)
	if nil != err {
		return err
	}
	sub.socket = socket

	// ***** FIX THIS ****
	// this allows any client to connect
	zmq.AuthAllow(submissionZapDomain, "127.0.0.1/8")
	zmq.AuthCurveAdd(submissionZapDomain, zmq.CURVE_ALLOW_ANY)

	// domain is servers public key
	socket.SetCurveServer(1)
	//socket.SetCurvePublickey(publicKey)
	socket.SetCurveSecretkey(privateKey)
	log.Infof("server public:  %q", publicKey)
	log.Infof("server private: %q", privateKey)

	socket.SetZapDomain(submissionZapDomain)

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
	for i, address := range configuration.Submit {
		bindTo, err := util.CanonicalIPandPort("tcp://", address)
		if nil != err {
			log.Errorf("submission[%d]=%q  error: %v", i, address, err)
			return err
		}

		err = socket.Bind(bindTo)
		if nil != err {
			log.Errorf("submit[%d]=%q  error: %v", i, address, err)
			socket.Close()
			return err
		}
		log.Infof("submit on: %q", address)
	}
	return nil
}

// wait for new blocks or new payment items
// to ensure the queue integrity as heap is not thread-safe
func (sub *submission) Run(args interface{}, shutdown <-chan struct{}) {

	log := sub.log

	log.Info("starting…")

	go func() {
		poller := zmq.NewPoller()
		poller.Add(sub.socket, zmq.POLLIN)
		poller.Add(sub.pull, zmq.POLLIN)
	loop:
		for {
			sockets, _ := poller.Poll(-1)
			for _, socket := range sockets {
				switch s := socket.Socket; s {
				case sub.socket:
					sub.process()
				case sub.pull:
					s.Recv(0)
					break loop
				}
			}
		}
		sub.pull.Close()
		sub.socket.Close()
	}()

	// wait for shutdown
	log.Info("waiting…")
	<-shutdown
	sub.push.SendMessage("stop")
	sub.push.Close()
}

// process the request and return response to prooferd
func (sub *submission) process() {

	log := sub.log

	data, err := sub.socket.RecvMessage(0)
	if nil != err {
		log.Errorf("JSON encode error: %v", err)
		return
	}

	log.Infof("received message: %q", data)

	// var request interface{}
	// err = json.Unmarshal([]byte(data), &request)
	// if nil != err {
	// 	log.Errorf("JSON decode error: %v", err)
	// 	continue
	// }

	// log.Infof("received message: %v", request)
	n := 1234

	response := struct {
		N  int
		OK bool
	}{
		N:  n,
		OK: true,
	}

	result, err := json.Marshal(response)
	if nil != err {
		log.Errorf("JSON encode error: %v", err)
		return
	}
	log.Infof("json to send: %s\n", result)

	// if _, err := socket.Send(to, zmq.SNDMORE|zmq.DONTWAIT); nil != err {
	// 	return err
	// }
	// if _, err := socket.Send(command, zmq.SNDMORE|zmq.DONTWAIT); nil != err {
	// 	return err
	// }
	_, err = sub.socket.SendBytes(result, 0|zmq.DONTWAIT)
	fault.PanicIfError("Submission", err)
}
