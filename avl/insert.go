// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package avl

// insert a new node into the tree
// returns the possibly updated root
func (tree *Tree) Insert(key item, value interface{}) bool {
	added := false
	tree.root, added, _ = insert(key, value, tree.root)
	if added {
		tree.count += 1
	}
	return added
}

// internal routine for insert
func insert(key item, value interface{}, p *Node) (*Node, bool, bool) {
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

					p1.up = p.up
					p.up = p1
					if nil != p.left {
						p.left.up = p
					}

					p = p1
				} else { // double LR rotation
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
			} else {
				// balance = +1, rebalance
				p1 := p.right
				if 1 == p1.balance {
					// single RR rotation
					p.right = p1.left
					p1.left = p
					p.balance = 0

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
