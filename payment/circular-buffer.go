// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package payment

import (
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/fault"
	"sync"
)

// node storage
type circularNode struct {
	next *circularNode
	list []string
}

// circular buffer for addresses
type circular struct {
	sync.Mutex // to allow locking need full mutex for hashtable - cannot use RWMutex

	node *circularNode
	set  map[string]int
}

// build circular buffer
func newCircular(size int) *circular {
	if size <= 0 {
		fault.Panic("negative or zero circular buffer size")
	}
	c1 := new(circularNode)
	c1.list = nil

	start := c1

	for i := 1; i < size; i += 1 {
		c2 := new(circularNode)
		c2.list = nil
		c2.next = c1
		c1 = c2
	}
	start.next = c1

	return &circular{
		node: start,
		set:  make(map[string]int, size),
	}
}

// break the circle so garbage collector can reclain the storage
func (c *circular) destroy() {
	c.Lock()
	defer c.Unlock()

	// already destroyed
	if nil == c.node {
		return
	}

	start := c.node
	c.set = nil
	c.node = nil
	for nil != start {
		next := start.next
		start.next = nil
		start = next
	}
}

// add a set of addresses to buffer overwriting the oldest one
func (c *circular) put(addresses []block.MinerAddress) {
	c.Lock()
	defer c.Unlock()

	if nil == c.node {
		fault.Panic("nil circular buffer node")
	}

	// remove old data
	for _, l := range c.node.list {
		count, ok := c.set[l]
		if ok {
			if count <= 1 {
				delete(c.set, l)
			} else {
				c.set[l] = count - 1
			}
		}
	}

	// add new data
	c.node.list = make([]string, len(addresses))
	for i, a := range addresses {
		currencyAddress := a.String()
		c.node.list[i] = currencyAddress
		c.set[currencyAddress] += 1
	}

	// move pointer
	c.node = c.node.next
}

// check if a particular address is present in the buffer
func (c *circular) isPresent(address block.MinerAddress) bool {
	c.Lock()
	defer c.Unlock()

	_, present := c.set[address.String()]
	return present
}
