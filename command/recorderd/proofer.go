// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/exitwithstatus"
	"github.com/bitmark-inc/logger"
	zmq "github.com/pebbe/zmq4"
)

const (
	proofRequest        = "inproc://blocks.request"  // to fair-queue block requests
	dispatch            = "inproc://blocks.dispatch" // proofer fetches from here
	errorProoferID      = -1
	prooferLoggerPrefix = "proofer"
)

var (
	proofQueueDepth uint64
)

type Proofer interface {
	StartHashing()
	StopHashing()
	IsWorking() bool
	Refresh()
}

type ProoferData struct {
	mu                    sync.RWMutex
	eventuallyThreadCount uint32
	prevThreadCount       uint32
	proofIDs              []bool
	stopChannel           chan struct{}
	log                   *logger.L
	workingNow            bool
	cpuCount              int
	reader                ConfigReader
}

func newProofer(log *logger.L, reader ConfigReader) Proofer {
	cpuCount := runtime.NumCPU()
	return &ProoferData{
		log:         log,
		proofIDs:    make([]bool, cpuCount),
		stopChannel: make(chan struct{}, cpuCount),
		cpuCount:    cpuCount,
		workingNow:  true,
		reader:      reader,
	}
}

func (p *ProoferData) StartHashing() {
	p.log.Infof("receive start hashing request, current active thread %d",
		p.targetThreadCount())
	p.setWorking(true)
	if p.targetThreadCount() < 1 {
		p.createProofer(p.reader.OptimalThreadCount())
	}
}

func (p *ProoferData) StopHashing() {
	p.log.Infof("receive stop hashing request, current active thread %d",
		p.targetThreadCount())
	p.setWorking(false)
	p.deleteProofer(int32(p.targetThreadCount()))
}

func (p *ProoferData) deleteProofer(count int32) {
	p.log.Infof("delete %d goroutine from hashing", count)
	for i := int32(0); i < count; i++ {

		p.eventuallyThreadCount--
		p.log.Debug("send signal to stop channel")
		p.stopChannel <- struct{}{}
	}
}

func (p *ProoferData) IsWorking() bool {
	return p.workingNow
}

func (p *ProoferData) setWorking(working bool) {
	p.workingNow = working
}

func (p *ProoferData) Refresh() {
	p.log.Infof("goroutine active count: %d, target count: %d",
		p.targetThreadCount(),
		p.reader.OptimalThreadCount(),
	)

	p.log.Infof("proofer setting change: %t, workable: %t",
		p.changed(),
		p.IsWorking(),
	)

	if !p.changed() || !p.IsWorking() {
		return
	}

	increment := p.differenceToTargetThreadCount(
		p.reader.OptimalThreadCount(),
		p.targetThreadCount(),
	)

	p.log.Infof("refresh settings, active goroutine %d, increase %d goroutine from hashing",
		p.targetThreadCount(), increment)

	if increment > 0 {
		p.createProofer(uint32(increment))
		return
	}
	p.deleteProofer(-increment)
}

func (p *ProoferData) changed() bool {
	return p.prevThreadCount != p.reader.OptimalThreadCount()
}

func (p *ProoferData) activeThreadIncrement(threadNum uint32) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.proofIDs[threadNum] = true
}

func (p *ProoferData) activeThreadDecrement(threadNum uint32) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.proofIDs[threadNum] = false
}

func (p *ProoferData) targetThreadCount() uint32 {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.eventuallyThreadCount
}

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
	if err != nil {
		return err
	}
	defer in.Close()

	in.SetLinger(0)
	err = in.Bind(proofRequest)
	if err != nil {
		return err
	}

	out, err := zmq.NewSocket(zmq.PUSH)
	if err != nil {
		return err
	}
	defer out.Close()

	_ = out.SetLinger(0)
	err = out.Bind(dispatch)
	if err != nil {
		return err
	}

	// possibly use this: ProxySteerable(frontend, backend, capture, control *Socket) error
	// with a control socket for clean shutdown
	return zmq.Proxy(in, out, nil)
}

func (p *ProoferData) nextProoferID() (int, error) {
	var idx int
	found := false

loop:
	for k, v := range p.proofIDs {
		if !v {
			idx = k
			found = true
			break loop
		}
	}

	if !found {
		return errorProoferID, fmt.Errorf("all proofer are used, abort")
	}
	return idx, nil
}

func (p *ProoferData) createProofer(threadCount uint32) {
	p.log.Infof("increase %d goroutine for hashing", threadCount)
	for i := uint32(0); i < threadCount; i++ {
		p.eventuallyThreadCount++
		proofID, err := p.nextProoferID()
		if err != nil {
			return
		}
		prflog := logger.New(fmt.Sprintf("proofer-%d", proofID))
		prflog.Infof("add new goroutine (%d out of this round increament %d)",
			i+1, threadCount)
		err = p.ProofThread(prflog, uint32(proofID))
		if err != nil {
			prflog.Criticalf("proof[%d]: error: %s", proofID, err)
			exitwithstatus.Message("proofer: proof[%d]: error: %s", proofID, err)
		}
	}
}

func (p *ProoferData) differenceToTargetThreadCount(
	targetThreadCount,
	currentThreadCount uint32,
) int32 {
	difference := int32(targetThreadCount) - int32(currentThreadCount)

	if math.Abs(float64(difference)) < math.Abs(float64(p.cpuCount)) {
		return int32(difference)
	}

	if targetThreadCount > currentThreadCount {
		return int32(p.cpuCount)
	}
	return int32(-p.cpuCount + 1)
}

func (p *ProoferData) ProofThread(log *logger.L, threadNum uint32) error {

	log.Infof("starting goroutine %d…", threadNum)

	// block request channel
	request, err := zmq.NewSocket(zmq.PULL)
	if err != nil {
		return err
	}

	request.SetLinger(0)
	err = request.Connect(dispatch)
	if err != nil {
		request.Close()
		return err
	}

	submit, err := zmq.NewSocket(zmq.PUSH)
	if err != nil {
		request.Close()
		return err
	}

	submit.SetLinger(0)
	err = submit.Connect(submission)
	if err != nil {
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

	p.activeThreadIncrement(threadNum)

	// background process
	go func() {
		defer request.Close()
		defer p.activeThreadDecrement(threadNum)

	receiver:
		for {
			request, err := request.RecvMessageBytes(0)
			if err != nil {
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
			json.Unmarshal(block, &item)
			log.Infof("received item: %v", item)

			// attempt to determine nonce
			timeout := time.After(MaximumSeconds)
			start := time.Now()
			count := 0
			blk := item.Header
		nonceLoop:
			for i := 0; true; i++ {
				select {
				case <-timeout:
					break nonceLoop
				case <-p.stopChannel:
					log.Infof("proofer %d receive stop event, terminate", threadNum)
					break receiver
				default:
					readyList, _ := poller.Poll(0) // time.Millisecond)
					//log.Infof("ready list: %v", readyList)
					//log.Infof("ready list length: %d", len(readyList))
					if len(readyList) == 1 {
						log.Info("new request, break nonceLoop")
						break nonceLoop
					}
				}

				// adjust Nonce, and compute new digest
				blk.Nonce++
				packed := blk.Pack()
				digest := blockdigest.NewDigest(packed[:])

				count++

				if i%10 == 0 {
					log.Infof("nonce[%d]: 0x%08x", i, blk.Nonce)
				}
				// possible value if leading zero byte
				if digest[31] == 0 {

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
