// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package announce

import (
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/pool"
	"sync"
)

// global data
var announce struct {
	sync.Mutex
	peerPool        *pool.IndexedPool
	rpcPool         *pool.IndexedPool
	certificatePool *pool.Pool
}

// initialise the pools
func Initialise() {
	announce.Lock()
	defer announce.Unlock()

	if nil != announce.peerPool || nil != announce.rpcPool || nil != announce.certificatePool {
		fault.Panic("announce.Initialise - already done")
	}
	announce.peerPool = pool.NewIndexed(pool.Peers)
	announce.rpcPool = pool.NewIndexed(pool.RPCs)
	announce.certificatePool = pool.New(pool.Certificates)
}

// close the pools
func Finalise() {
	announce.Lock()
	defer announce.Unlock()

	if nil == announce.peerPool || nil == announce.rpcPool || nil == announce.certificatePool {
		return
	}

	announce.peerPool.Flush()
	announce.rpcPool.Flush()
	announce.certificatePool.Flush()

	announce.peerPool = nil
	announce.rpcPool = nil
	announce.certificatePool = nil
}
