// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package proof

import (
	"encoding/json"

	"github.com/bitmark-inc/bitmarkd/p2p"

	zmq "github.com/pebbe/zmq4"

	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/logger"
)

const (
	submissionZapDomain = "submit"
	submissionSignal    = "inproc://bitmark-submission-signal"
)

type submission struct {
	log        *logger.L
	sigSend    *zmq.Socket // signal send
	sigReceive *zmq.Socket // signal receive
	socket4    *zmq.Socket
	socket6    *zmq.Socket
}

// initialise the submission
func (sub *submission) initialise(configuration *Configuration) error {

	log := logger.New("submission")
	sub.log = log

	log.Info("initialising…")

	// read the keys
	privateKey, err := zmqutil.ReadPrivateKey(configuration.PrivateKey)
	if nil != err {
		return err
	}
	publicKey, err := zmqutil.ReadPublicKey(configuration.PublicKey)
	if nil != err {
		return err
	}
	log.Tracef("server public:  %x", publicKey)
	log.Tracef("server private: %x", privateKey)

	// signalling channel
	sub.sigReceive, sub.sigSend, err = zmqutil.NewSignalPair(submissionSignal)
	if nil != err {
		return err
	}

	// create connections
	//c, _ := util.NewConnections(configuration.Submit)

	// allocate IPv4 and IPv6 sockets
	//sub.socket4, sub.socket6, err = zmqutil.NewBind(log, zmq.REP, submissionZapDomain, privateKey, publicKey, c)
	//if nil != err {
	//	log.Errorf("bind error: %s", err)
	//	return err
	//}

	return nil
}

// wait for new blocks or new payment items
// to ensure the queue integrity as heap is not thread-safe
func (sub *submission) Run(args interface{}, shutdown <-chan struct{}) {

	log := sub.log

	log.Info("starting…")

	go func() {
		//poller := zmq.NewPoller()
		//if nil != sub.socket4 {
		//	poller.Add(sub.socket4, zmq.POLLIN)
		//}
		//if nil != sub.socket6 {
		//	poller.Add(sub.socket6, zmq.POLLIN)
		//}
		//poller.Add(sub.sigReceive, zmq.POLLIN)
		//loop:
		//	for {
		//		log.Debug("waiting…")
		//		sockets, _ := poller.Poll(-1)
		//		for _, socket := range sockets {
		//			switch s := socket.Socket; s {
		//			case sub.sigReceive:
		//				s.Recv(0)
		//				break loop
		//			default:
		//				sub.process(s)
		//			}
		//		}
		//	}
		//	sub.sigReceive.Close()
		//if nil != sub.socket4 {
		//	sub.socket4.Close()
		//}
		//if nil != sub.socket6 {
		//	sub.socket6.Close()
		//}

		for j := range possibleHashCh {
			log.Debug("receive possible hash")
			sub.processP2P(j)
		}
	}()

	// wait for shutdown
	<-shutdown
	sub.sigSend.SendMessage("stop")
	sub.sigSend.Close()
}

// process the request and return response to prooferd
func (sub *submission) process(socket *zmq.Socket) {
	log := sub.log

	data, err := socket.RecvMessage(0)
	if nil != err {
		log.Errorf("JSON encode error: %s", err)
		return
	}

	log.Infof("received message: %q", data)

	ok := false
	var request SubmittedItem
	err = json.Unmarshal([]byte(data[0]), &request)
	if nil != err {
		log.Errorf("JSON decode error: %s", err)
	} else {

		log.Infof("received message: %v", request)

		ok = matchToJobQueue(&request, log)

		log.Infof("matches: %v", ok)
	}

	response := struct {
		Job string `json:"job"`
		OK  bool   `json:"ok"`
	}{
		Job: request.Job,
		OK:  ok,
	}

	result, err := json.Marshal(response)
	logger.PanicIfError("JSON encode error", err)

	log.Infof("json to send: %s", result)

	// if _, err := socket.Send(to, zmq.SNDMORE|zmq.DONTWAIT); nil != err {
	// 	return err
	// }
	// if _, err := socket.Send(command, zmq.SNDMORE|zmq.DONTWAIT); nil != err {
	// 	return err
	// }
	_, err = socket.SendBytes(result, 0|zmq.DONTWAIT)
	logger.PanicIfError("Submission", err)
}

func (sub *submission) processP2P(data []byte) {
	log := sub.log

	log.Infof("received message: %q", data)
	_, fn, parameters, err := p2p.UnPackP2PMessage(data)
	if nil != err || fn != "S" {
		log.Error("unpack received message error")
		return
	}

	ok := false
	var request SubmittedItem
	err = json.Unmarshal(parameters[0], &request)
	if nil != err {
		log.Errorf("JSON decode error: %s", err)
	} else {
		log.Infof("received message: %v", request)
		ok = matchToJobQueue(&request, log)
		log.Infof("matches: %v", ok)
	}

	response := struct {
		Job string `json:"job"`
		OK  bool   `json:"ok"`
	}{
		Job: request.Job,
		OK:  ok,
	}

	result, err := json.Marshal(response)
	logger.PanicIfError("JSON encode error", err)
	log.Infof("json hash result to send: %s", result)

	resultToSendCh <- result
}
