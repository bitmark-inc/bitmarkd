// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package avl

// Insert - insert a new node into the tree
// returns the possibly updated root
func (tree *Tree) Insert(key Item, value interface{}) bool {
	added := false
	tree.root, added, _ = insert(key, value, tree.root)
	if added {
		tree.count += 1
	}
	return added
}

// internal routine for insert
func insert(key Item, value interface{}, p *Node) (*Node, bool, bool) {
	h := false
	if nil == p { // insert new node
		h = true
		p = newNode(key, value)
		return p, true, h
	}
	added := false
	switch p.key.Compare(key) {
	case +1: // p.key > key
		p.left, added, h = insert(key, value, p.left)
		if added {
			p.leftNodes += 1
		}
		if h {
			if nil != p.left {
				p.left.up = p
			}

			// left branch has grown
			if 1 == p.balance {
				p.balance = 0
				h = false
			} else if 0 == p.balance {
				p.balance = -1
			} else { // balance == -1, rebalance
				p1 := p.left
				if -1 == p1.balance {
					// single LL rotation
					p.left = p1.right
					p1.right = p

					p.balance = 0

					nn := 1 + p1.rightNodes + p.rightNodes
					p.leftNodes = p1.rightNodes
					p1.rightNodes = nn

					p1.up = p.up
					p.up = p1
					if nil != p.left {
						p.left.up = p
					}

					p = p1
				} else {
					// double LR rotation
					p2 := p1.right
					p1.right = p2.left
					p2.left = p1
					p.left = p2.right
					p2.right = p
					if -1 == p2.balance {
						p.balance = 1
					} else {
						p.balance = 0
					}
					if +1 == p2.balance {
						p1.balance = -1
					} else {
						p1.balance = 0
					}

					nl := 1 + p1.leftNodes + p2.leftNodes
					nr := 1 + p2.rightNodes + p.rightNodes

					p1.rightNodes = p2.leftNodes
					p.leftNodes = p2.rightNodes

					p2.leftNodes = nl
					p2.rightNodes = nr

					if nil != p.left {
						p.left.up = p
					}
					if nil != p1.right {
						p1.right.up = p1
					}
					p2.up = p.up
					p.up = p2
					p1.up = p2

					p = p2
				}
				p.balance = 0
				h = false
			}
		}
	case -1: // p.key < key
		p.right, added, h = insert(key, value, p.right)
		if added {
			p.rightNodes += 1
		}
		if h {
			if nil != p.right {
				p.right.up = p
			}

			// right branch has grown
			if -1 == p.balance {
				p.balance = 0
				h = false
			} else if 0 == p.balance {
				p.balance = 1
			} else { // balance = +1, rebalance
				p1 := p.right
				if 1 == p1.balance {
					// single RR rotation
					p.right = p1.left
					p1.left = p

					p.balance = 0

					nn := 1 + p.leftNodes + p1.leftNodes
					p.rightNodes = p1.leftNodes
					p1.leftNodes = nn

					p1.up = p.up
					p.up = p1
					if nil != p.right {
						p.right.up = p
					}

					p = p1
				} else {
					// double RL rotation
					p2 := p1.left
					p1.left = p2.right
					p2.right = p1
					p.right = p2.left
					p2.left = p
					if +1 == p2.balance {
						p.balance = -1
					} else {
						p.balance = 0
					}
					if -1 == p2.balance {
						p1.balance = 1
					} else {
						p1.balance = 0
					}

					nl := 1 + p.leftNodes + p2.leftNodes
					nr := 1 + p2.rightNodes + p1.rightNodes

					p.rightNodes = p2.leftNodes
					p1.leftNodes = p2.rightNodes

					p2.leftNodes = nl
					p2.rightNodes = nr

					if nil != p.right {
						p.right.up = p
					}
					if nil != p1.left {
						p1.left.up = p1
					}
					p2.up = p.up
					p.up = p2
					p1.up = p2

					p = p2
				}
				p.balance = 0
				h = false
			}
		}
	default:
		p.value = value
	}
	return p, added, h
}
