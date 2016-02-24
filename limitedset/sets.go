// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package limitedset

import (
	"container/ring"
	"sync"
)

type LimitedSet struct {
	sync.Mutex
	size int
	ring *ring.Ring
	hash map[string]*ring.Ring
}

// create a new limited set that holds up to 'n' items
func New(n int) *LimitedSet {
	return &LimitedSet{
		size: n,
		ring: ring.New(n),
		hash: make(map[string]*ring.Ring),
	}
}

// add an item to the set
func (ls *LimitedSet) Add(item string) {
	ls.Lock()
	defer ls.Unlock()
	if r, ok := ls.hash[item]; ok {
		r = r.Prev().Unlink(1)
		ls.ring.Prev().Link(r)
		return
	}
	if oldItem, ok := ls.ring.Value.(string); ok {
		delete(ls.hash, oldItem)
	}
	ls.ring.Value = item
	ls.hash[item] = ls.ring
	ls.ring = ls.ring.Next()
}

// check to see if items is in the set
func (ls *LimitedSet) Exists(item string) bool {
	ls.Lock()
	defer ls.Unlock()
	_, ok := ls.hash[item]
	return ok
}
