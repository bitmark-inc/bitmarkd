// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"encoding/binary"
	"encoding/json"
	"sync/atomic"
	"time"

	zmq "github.com/pebbe/zmq4"

	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/logger"
)

const (
	proofRequest = "inproc://blocks.request"  // to fair-queue block requests
	dispatch     = "inproc://blocks.dispatch" // proofer fetches from here
)

var proofQueueDepth uint64

func ProofQueueIncrement() {
	atomic.AddUint64(&proofQueueDepth, 1)
}

func ProofQueueDecrement() {
	atomic.AddUint64(&proofQueueDepth, 0xffffffffffffffff)
}

// this provides a single submission point for hashing requests
// multiple proof threads can attach and fair queuing takes place
func ProofProxy() {
	go func() {
		err := proofForwarder()
		logger.PanicIfError("proofProxy", err)
	}()
}

// internal proxy forwarding loop
func proofForwarder() error {

	in, err := zmq.NewSocket(zmq.PULL)
	if nil != err {
		return err
	}
	defer in.Close()

	in.SetLinger(0)
	err = in.Bind(proofRequest)
	if nil != err {
		return err
	}

	out, err := zmq.NewSocket(zmq.PUSH)
	if nil != err {
		return err
	}
	defer out.Close()

	out.SetLinger(0)
	err = out.Bind(dispatch)
	if nil != err {
		return err
	}

	// possibly use this: ProxySteerable(frontend, backend, capture, control *Socket) error
	// with a control socket for clean shutdown
	return zmq.Proxy(in, out, nil)
}

// proof thread thread
func ProofThread(log *logger.L) error {

	log.Info("starting…")

	// block request channel
	request, err := zmq.NewSocket(zmq.PULL)
	if nil != err {
		return err
	}

	request.SetLinger(0)
	err = request.Connect(dispatch)
	if nil != err {
		request.Close()
		return err
	}

	submit, err := zmq.NewSocket(zmq.PUSH)
	if nil != err {
		request.Close()
		return err
	}

	submit.SetLinger(0)
	err = submit.Connect(submission)
	if nil != err {
		request.Close()
		submit.Close()
		return err
	}

	// go auth_do_handler()

	// // basic socket options
	// //socket.SetIpv6(true)  // ***** FIX THIS find fix for FreeBSD libzmq4 ****
	// socket.SetSndtimeo(SEND_TIMEOUT)
	// socket.SetLinger(LINGER_TIME)
	// socket.SetRouterMandatory(0)   // discard unroutable packets
	// socket.SetRouterHandover(true) // allow quick reconnect for a given public key
	// socket.SetImmediate(false)     // queue messages sent to disconnected peer

	poller := zmq.NewPoller()
	poller.Add(request, zmq.POLLIN)

	// background process
	go func() {
		defer request.Close()

	receiver:
		for {
			request, err := request.RecvMessageBytes(0)
			if nil != err {
				log.Criticalf("RecvMessageBytes error: %s", err)
				logger.Panicf("proofer error: %s", err)
			}

			ProofQueueDecrement()

			log.Infof("received data: %s", request)

			// flush short messages
			if len(request) < 2 {
				continue receiver
			}

			// split message request
			submitter := request[0]
			block := request[1]

			MaximumSeconds := 120 * time.Second

			var item PublishedItem
			err = json.Unmarshal(block, &item)
			log.Infof("received item: %v", item)

			// attempt to determine nonce
			timeout := time.After(MaximumSeconds)
			start := time.Now()
			count := 0
			blk := item.Header
		nonceLoop:
			for i := 0; true; i += 1 {

				select {
				case <-timeout:
					break nonceLoop
				default:
					readyList, _ := poller.Poll(0) // time.Millisecond)
					//log.Infof("ready list: %v", readyList)
					//log.Infof("ready list length: %d", len(readyList))
					if 1 == len(readyList) {
						log.Info("new request, break nonceLoop")
						break nonceLoop
					}
				}

				// adjust Nonce, and compute new digest
				blk.Nonce += 1
				packed := blk.Pack()
				digest := blockdigest.NewDigest(packed[:])

				count += 1

				if 0 == i%10 {
					log.Infof("nonce[%d]: 0x%08x", i, blk.Nonce)
				}
				// possible value if leading zero byte
				if 0 == digest[31] {

					log.Infof("job: %q nonce: 0x%016x", item.Job, blk.Nonce)
					log.Infof("digest: %v", digest)

					nonce := make([]byte, blockrecord.NonceSize)
					binary.LittleEndian.PutUint64(nonce, uint64(blk.Nonce))

					_, err := submit.SendBytes(submitter, zmq.SNDMORE) // routing address
					logger.PanicIfError("submit send", err)
					_, err = submit.SendBytes(submitter, zmq.SNDMORE) // destination check
					logger.PanicIfError("submit send", err)
					_, err = submit.Send(item.Job, zmq.SNDMORE) // job id
					logger.PanicIfError("submit send", err)
					_, err = submit.SendBytes(nonce, 0) // actual data
					logger.PanicIfError("submit send", err)

					// ************** if actual difficulty is met
					// if ... { break nonceLoop }
				}
			}

			// compute hash rate
			rate := float64(count) / time.Since(start).Minutes()
			log.Infof("hash rate: %f H/min", rate)
		}
	}()
	return nil
}
