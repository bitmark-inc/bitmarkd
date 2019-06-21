// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package avl

import (
	"sync"
)

// Item - a key item must implement the Compare function
type Item interface {
	Compare(interface{}) int // for left/right ordering of items
}

// Node - a node in the tree
type Node struct {
	left       *Node       // left sub-tree
	right      *Node       // right sub-tree
	up         *Node       // points to parent node
	key        Item        // key part for ordering
	value      interface{} // value part for data storage
	leftNodes  int         // node count
	rightNodes int         // node count
	balance    int         // -1, 0, +1
}

// global data for allocator
var m sync.Mutex   // to keep values in sync
var pool *Node     // linked list of reclaimed nodes
var totalNodes int // total nodes created
var freeNodes int  // number of nodes in the pool

// allocate a new node, reuses reclaimed nodes if any are available
func newNode(key Item, value interface{}) *Node {
	m.Lock()
	if nil == pool {
		if 0 != freeNodes {
			panic("pool corrupt")
		}
		totalNodes += 1
		m.Unlock()
		return &Node{
			key:     key,
			value:   value,
			balance: 0,
		}
	}
	p := pool
	pool = p.up
	p.key = key
	p.value = value
	p.balance = 0
	p.leftNodes = 0
	p.rightNodes = 0
	p.left = nil
	p.right = nil
	p.up = nil // ensure freelist pointer is cleared
	freeNodes -= 1
	m.Unlock()
	return p
}

// reclaim a node and keep it in a pool
func freeNode(node *Node) {
	m.Lock()
	node.up = pool // use as free list pointer

	node.left = nil
	node.right = nil
	node.key = nil
	node.value = nil
	node.balance = 0
	node.leftNodes = 0
	node.rightNodes = 0
	freeNodes += 1

	pool = node
	m.Unlock()
}
