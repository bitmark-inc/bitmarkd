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
	submissionSignal = "inproc://bitmark-submission-signal"
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

	return nil
}

// wait for new blocks or new payment items
// to ensure the queue integrity as heap is not thread-safe
func (sub *submission) Run(args interface{}, shutdown <-chan struct{}) {

	log := sub.log

	log.Info("starting…")

	go func() {
		for hash := range possibleHashCh {
			log.Debug("receive possible hash")
			sub.process(hash)
		}
	}()

	// wait for shutdown
	<-shutdown
	_, _ = sub.sigSend.SendMessage("stop")
	_ = sub.sigSend.Close()
}

// process the request and return response to recorderd
func (sub *submission) process(data []byte) {
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
