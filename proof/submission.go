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
	log     *logger.L
	push    *zmq.Socket
	pull    *zmq.Socket
	socket4 *zmq.Socket
	socket6 *zmq.Socket
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
	privateKey, err := zmqutil.ReadPrivateKeyFile(configuration.PrivateKey)
	if nil != err {
		return err
	}
	publicKey, err := zmqutil.ReadPublicKeyFile(configuration.PublicKey)
	if nil != err {
		return err
	}
	log.Tracef("server public:  %x", publicKey)
	log.Tracef("server private: %x", privateKey)

	// signalling channel
	sub.push, sub.pull, err = zmqutil.NewSignalPair(submissionSignal)
	if nil != err {
		return err
	}

	// create connections
	c, err := util.NewConnections(configuration.Submit)

	// allocate IPv4 and IPv6 sockets
	sub.socket4, sub.socket6, err = zmqutil.NewBind(log, zmq.REP, submissionZapDomain, privateKey, publicKey, c)
	if nil != err {
		log.Errorf("bind error: %v", err)
		return err
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
		if nil != sub.socket4 {
			poller.Add(sub.socket4, zmq.POLLIN)
		}
		if nil != sub.socket6 {
			poller.Add(sub.socket6, zmq.POLLIN)
		}
		poller.Add(sub.pull, zmq.POLLIN)
	loop:
		for {
			log.Info("waiting…")
			sockets, _ := poller.Poll(-1)
			for _, socket := range sockets {
				switch s := socket.Socket; s {
				case sub.pull:
					s.Recv(0)
					break loop
				default:
					sub.process(s)
				}
			}
		}
		sub.pull.Close()
		if nil != sub.socket4 {
			sub.socket4.Close()
		}
		if nil != sub.socket6 {
			sub.socket6.Close()
		}
	}()

	// wait for shutdown
	<-shutdown
	sub.push.SendMessage("stop")
	sub.push.Close()
}

// process the request and return response to prooferd
func (sub *submission) process(socket *zmq.Socket) {

	log := sub.log

	data, err := socket.RecvMessage(0)
	if nil != err {
		log.Errorf("JSON encode error: %v", err)
		return
	}

	log.Infof("received message: %q", data)

	var request SubmittedItem
	err = json.Unmarshal([]byte(data[0]), &request)
	if nil != err {
		log.Errorf("JSON decode error: %v", err)
	}

	log.Infof("received message: %v", request)

	ok := matchToJobQueue(&request)

	log.Infof("maches: %v", ok)

	response := struct {
		Job string `json:"job"`
		OK  bool   `json:"ok"`
	}{
		Job: request.Job,
		OK:  ok,
	}

	result, err := json.Marshal(response)
	if nil != err {
		log.Errorf("JSON encode error: %v", err)
		return
	}
	log.Infof("json to send: %s", result)

	// if _, err := socket.Send(to, zmq.SNDMORE|zmq.DONTWAIT); nil != err {
	// 	return err
	// }
	// if _, err := socket.Send(command, zmq.SNDMORE|zmq.DONTWAIT); nil != err {
	// 	return err
	// }
	_, err = socket.SendBytes(result, 0|zmq.DONTWAIT)
	fault.PanicIfError("Submission", err)
}
