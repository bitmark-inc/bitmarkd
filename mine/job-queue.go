// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package mine

import (
	"encoding/hex"
	"fmt"
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/mode"
	"sync"
	"time"
)

// limits
const (
	// assuming a new job every 10 seconds and a maximum block time of 15 minutes
	queueSize = 15 * 60 / 10

	// this jobId will never occur
	// special code in add to prevent this from happening
	jobIdentifierNil = jobIdentifier(0)
)

// type for job
type jobIdentifier uint16 // bigger than index to account for reallocations

// a job
type job struct {
	jobId     jobIdentifier
	ids       []block.Digest
	addresses []block.MinerAddress
	timestamp time.Time
	accessed  bool
}

// the job queue
type queue struct {
	sync.RWMutex

	jobIdAllocator jobIdentifier          // only incremented, type is bigger that queue index to avoid clash
	topIndex       int                    // head of queue
	startTime      time.Time              // time of first entry (only set after queue was cleared)
	jobs           [queueSize]job         // array of jobs
	topJob         *job                   // fast access to top item
	index          map[jobIdentifier]*job // index of active items
}

// the master queue
var jobQueue *queue

// initialise the background process
func initialiseJobQueue() {

	// set up queue
	jobQueue = &queue{
		jobIdAllocator: 0,
		topIndex:       0,
		topJob:         nil,
		index:          make(map[jobIdentifier]*job),
	}
}

// save the ids
//
// need to save a copy of address in case some other part changes it
// this will overwrite the current entry if it was not used
func (queue *queue) add(ids []block.Digest, addresses []block.MinerAddress, timestamp time.Time) {

	// skip empty jobs
	if 0 == len(ids) {
		return
	}

	// if suspended no new jobs are allowed
	if mode.IsNot(mode.Normal) {
		return
	}

	// get write access
	queue.Lock()
	defer queue.Unlock()

	// safe since write locked
	// increment allocator - just allow to wrap within type size
	// Important: ensure the nil value cannot occur
	queue.jobIdAllocator += 1
	if jobIdentifierNil == queue.jobIdAllocator {
		queue.jobIdAllocator += 1
	}

	// reference to to item
	p := &queue.jobs[queue.topIndex]

	// if accessed then need to use / overwrite next element
	if p.accessed {
		// increment and wrap
		next := queue.topIndex + 1
		if next >= queueSize {
			fault.Criticalf("queueForMining: queue full")
			fault.Panic("queueForMining: queue full")

			next = 0
		}
		queue.topIndex = next
		p = &queue.jobs[queue.topIndex]
	}

	// remove old index
	delete(queue.index, p.jobId)

	// start out not accessed
	p.accessed = false

	// new id
	p.jobId = queue.jobIdAllocator

	// copy the id list
	p.ids = make([]block.Digest, len(ids))
	copy(p.ids, ids)

	// copy the addresses
	p.addresses = make([]block.MinerAddress, len(addresses))
	copy(p.addresses, addresses)

	// store timestamp
	p.timestamp = timestamp

	// index the entry for later recall
	queue.index[queue.jobIdAllocator] = p
	if nil == queue.topJob {
		queue.startTime = timestamp
	}
	queue.topJob = p
}

// fetch the min tree for and the latest job id for miner notify
func (queue *queue) top() (jobId jobIdentifier, mintree []block.Digest, adddresses []block.MinerAddress, timestamp time.Time, clean bool, ok bool) {
	queue.RLock()
	defer queue.RUnlock()

	// suspended or empty
	if mode.IsNot(mode.Normal) || nil == queue.topJob {
		return 0, nil, nil, time.Time{}, false, false
	}

	// retrieve top job and mark accessed
	topJob := queue.topJob
	topJob.accessed = true

	if nil == topJob.ids {
		return 0, nil, nil, time.Time{}, false, false
	}

	minTree := block.MinimumMerkleTree(topJob.ids)

	//return topJob.jobId, minTree, topJob.addresses, topJob.timestamp, 0 == queue.topIndex, true
	return topJob.jobId, minTree, topJob.addresses, queue.startTime, 0 == queue.topIndex, true
}

// get a list of transaction ids
func (queue *queue) getIds(jobId jobIdentifier) (ids []block.Digest, addresses []block.MinerAddress, timestamp time.Time, valid bool) {
	queue.RLock()
	defer queue.RUnlock()

	// attempt to access a job
	job, ok := queue.index[jobId]

	// ensure entry is not expired
	if mode.IsNot(mode.Normal) || !ok || job.jobId != jobId {
		return nil, nil, time.Time{}, false // fail if trying to confirm expired entry
	}

	//return job.ids, job.addresses, job.timestamp, true
	return job.ids, job.addresses, queue.startTime, true
}

// job was mined sucessfully
func (queue *queue) confirm(jobId jobIdentifier) []block.Digest {
	queue.Lock()
	defer queue.Unlock()

	job, ok := queue.index[jobId]

	// ensure entry/entire queue is not expired
	if mode.IsNot(mode.Normal) || !ok || job.jobId != jobId {
		return nil // fail if trying to confirm expired entry
	}

	// invalidate all entries in queue
	queue.doClear()

	return job.ids
}

func (queue *queue) doClear() {
	// *** same code as clear ***

	// invalidate all older jobs - by just creating a new empty table
	// and waiting for it to be garbage collected
	queue.index = make(map[jobIdentifier]*job)

	// empty the queue
	queue.topIndex = 0
	queue.topJob = nil
	queue.jobs[0].accessed = false
}

// abandon all jobs and clear the queue
//
// note queue can be cleared in suspend state
func (queue *queue) clear() {
	queue.Lock()
	defer queue.Unlock()

	// invalidate all entries in queue
	queue.doClear()
}

// see if queue is clear
func (queue *queue) isClear() bool {
	queue.RLock()
	defer queue.RUnlock()

	return nil == queue.topJob
}

// convert a job id into a string
func (jobId jobIdentifier) String() string {
	return fmt.Sprintf("%04x", uint16(jobId))
}

// convert a string into a job id any error just returns the zero value
func stringToJobId(s string) jobIdentifier {
	h, err := hex.DecodeString(s)
	if nil != err {
		return 0
	}
	if 2 != len(h) {
		return 0
	}
	// big endian
	return jobIdentifier(h[1]) + 256*jobIdentifier(h[0])
}
