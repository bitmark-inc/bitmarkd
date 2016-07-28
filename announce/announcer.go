// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package announce

import (
	"github.com/bitmark-inc/bitmarkd/fault"
	//"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/logger"
	"time"
)

const (
	announceInitial  = 2 * time.Minute
	announceInterval = 10 * time.Minute
	announceExpiry   = 60 * time.Minute
)

type announcer struct {
	log *logger.L
}

// initialise the announcer
func (ann *announcer) initialise() error {

	log := logger.New("announcer")
	if nil == log {
		return fault.ErrInvalidLoggerChannel
	}
	ann.log = log

	log.Info("initialising…")

	return nil
}

// wait for incoming requests, process them and reply
func (ann *announcer) Run(args interface{}, shutdown <-chan struct{}) {

	log := ann.log

	log.Info("starting…")

	delay := time.After(announceInitial)
loop:
	for {
		log.Info("waiting…")
		select {
		case <-shutdown:
			break loop
		case <-delay:
			delay = time.After(announceInterval)
			ann.process()
		}
	}
}

// process the ann and return response to client
func (ann *announcer) process() {

	log := ann.log

	log.Info("process starting…")

	// announce this nodes IP and ports to other peers
	// for _, rpc := range globalData.rpcs {
	// 	messagebus.Send("rpc", rpc.fingerprint[:], rpc.address.Pack())
	// }
	// for _, broadcast := range globalData.broadcasts {
	// 	messagebus.SendString("broadcast", broadcast)
	// }
	// for _, listener := range globalData.listeners {
	// 	messagebus.SendString("listener", listener)
	// }
}
