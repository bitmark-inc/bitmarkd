// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"

	zmq "github.com/pebbe/zmq4"

	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/logger"
)

// sent by bitmarkd
// ***** FIX THIS: need to refactor
type PublishedItem struct {
	Job    string
	Header blockrecord.Header
}

func Subscriber(
	i int,
	log *logger.L,
	proofer Proofer,
	hashRequestChan <-chan []byte,
) error {

	log.Info("starting…")

	identity := fmt.Sprintf("subscriber-%d", i)
	mySubmitterIdentity := fmt.Sprintf("submitter-%d", i) // ***** FIX THIS: sync up with submitter so names match *****

	// to submit hashing requests
	proof, err := zmq.NewSocket(zmq.PUSH)
	if nil != err {
		return err
	}

	_ = proof.SetLinger(0)
	_ = proof.SetIdentity(identity)
	err = proof.Connect(proofRequest)
	if nil != err {
		_ = proof.Close()
	}

	// background process
	go func() {
		defer proof.Close()

	loop:
		for {
			select {
			case data := <-hashRequestChan:
				log.Infof("receive hash request chan")
				// prevent queuing outdated request
				if !proofer.IsWorking() {
					log.Infof("Rest time, discard request")
					continue loop
				}

				// ***** FIX THIS: just debugging? or really split block into multiple nonce ranges
				var item PublishedItem
				err = json.Unmarshal([]byte(data), &item)
				if nil != err {
					log.Errorf("unmarshal json %v with error: %s", data, err)
					continue
				}
				log.Infof("unmarshal received: %v", item)

				// initial try just forward block
				_, err = proof.Send(mySubmitterIdentity, zmq.SNDMORE)
				logger.PanicIfError("subscriber sending 1", err)
				_, err = proof.Send(string(data), 0)
				logger.PanicIfError("subscriber sending 2", err)
				ProofQueueIncrement()
				log.Infof("queue depth: %d", proofQueueDepth)
			}
		}
	}()
	return nil
}
