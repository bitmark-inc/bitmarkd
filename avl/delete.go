// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package avl

// delete: tree balancer
func balanceLeft(pp **Node) bool {
	h := true
	p := *pp
	// h; left branch has shrunk
	if p.balance == -1 {
		p.balance = 0
	} else if p.balance == 0 {
		p.balance = 1
		h = false
	} else { // balance = 1, rebalance
		p1 := p.right
		if p1.balance >= 0 {
			// single RR rotation
			p.right = p1.left
			p1.left = p
			if p1.balance == 0 {
				p.balance = 1
				p1.balance = -1
				h = false
			} else {
				p.balance = 0
				p1.balance = 0
			}

			nn := 1 + p.leftNodes + p1.leftNodes
			p.rightNodes = p1.leftNodes
			p1.leftNodes = nn

			p1.up = p.up
			p.up = p1
			if p.right != nil {
				p.right.up = p
			}

			*pp = p1
		} else {
			// double RL rotation
			p2 := p1.left
			p1.left = p2.right
			p2.right = p1
			p.right = p2.left
			p2.left = p
			if p2.balance == +1 {
				p.balance = -1
			} else {
				p.balance = 0
			}
			if p2.balance == -1 {
				p1.balance = 1
			} else {
				p1.balance = 0
			}
			p2.balance = 0

			nl := 1 + p.leftNodes + p2.leftNodes
			nr := 1 + p2.rightNodes + p1.rightNodes

			p.rightNodes = p2.leftNodes
			p1.leftNodes = p2.rightNodes

			p2.leftNodes = nl
			p2.rightNodes = nr

			p2.up = p.up
			if p.right != nil {
				p.right.up = p
			}
			if p1.left != nil {
				p1.left.up = p1
			}
			p.up = p2
			p1.up = p2

			*pp = p2
		}
	}
	return h
}

// delete: tree balancer
func balanceRight(pp **Node) bool {
	h := true
	p := *pp
	// h; right branch has shrunk
	if p.balance == 1 {
		p.balance = 0
	} else if p.balance == 0 {
		p.balance = -1
		h = false
	} else { // balance = -1, rebalance
		p1 := p.left
		if p1.balance <= 0 {
			// single LL rotation
			p.left = p1.right
			p1.right = p
			if p1.balance == 0 {
				p.balance = -1
				p1.balance = 1
				h = false
			} else {
				p.balance = 0
				p1.balance = 0
			}

			nn := 1 + p1.rightNodes + p.rightNodes
			p.leftNodes = p1.rightNodes
			p1.rightNodes = nn

			p1.up = p.up
			p.up = p1
			if p.left != nil {
				p.left.up = p
			}

			*pp = p1
		} else {
			// double LR rotation
			p2 := p1.right
			p1.right = p2.left
			p2.left = p1
			p.left = p2.right
			p2.right = p
			if p2.balance == -1 {
				p.balance = 1
			} else {
				p.balance = 0
			}
			if p2.balance == +1 {
				p1.balance = -1
			} else {
				p1.balance = 0
			}
			p2.balance = 0

			nl := 1 + p1.leftNodes + p2.leftNodes
			nr := 1 + p2.rightNodes + p.rightNodes

			p1.rightNodes = p2.leftNodes
			p.leftNodes = p2.rightNodes

			p2.leftNodes = nl
			p2.rightNodes = nr

			p2.up = p.up
			if p.left != nil {
				p.left.up = p
			}
			if p1.right != nil {
				p1.right.up = p1
			}
			p.up = p2
			p1.up = p2

			*pp = p2
		}
	}
	return h
}

// delete: rearrange deleted node
func del(qq **Node, rr **Node) bool {
	h := false
	if (*rr).right != nil {
		h = del(qq, &(*rr).right)
		(*rr).rightNodes -= 1
		if h {
			h = balanceRight(rr)
		}
	} else {
		q := *qq
		r := *rr
		rl := r.left
		if rl != nil {
			rl.up = r.up
		}

		if r != q.left {
			r.left = q.left
			r.leftNodes = q.leftNodes - 1
		}
		r.right = q.right
		r.up = q.up
		r.balance = q.balance
		r.rightNodes = q.rightNodes

		if r.right != nil {
			r.right.up = r
		}
		if r.left != nil {
			r.left.up = r
		}

		*qq = r
		*rr = rl

		h = true
	}
	return h
}

// Delete - removes a specific item from the tree
func (tree *Tree) Delete(key Item) interface{} {
	value, removed, _ := intdelete(key, &tree.root)
	if removed {
		tree.count -= 1
	}
	return value
}

// internal delete routine
func intdelete(key Item, pp **Node) (interface{}, bool, bool) {
	h := false
	if *pp == nil { // key not in tree
		return nil, false, h
	}
	value := interface{}(nil)
	removed := false
	switch (*pp).key.Compare(key) {
	case +1: // (*pp).key > key
		value, removed, h = intdelete(key, &(*pp).left)
		if removed {
			(*pp).leftNodes -= 1
		}
		if h {
			h = balanceLeft(pp)
		}
	case -1: // (*pp).key < key
		value, removed, h = intdelete(key, &(*pp).right)
		if removed {
			(*pp).rightNodes -= 1
		}
		if h {
			h = balanceRight(pp)
		}
	default: // found: delete p
		q := *pp
		value = q.value // preserve the value part
		if q.right == nil {
			if q.left != nil {
				q.left.up = q.up
			}
			*pp = q.left
			h = true
		} else if q.left == nil {
			if q.right != nil {
				q.right.up = q.up
			}
			*pp = q.right
			h = true
		} else {
			h = del(pp, &q.left)
			(*pp).left = q.left // p has changed, but q.left has left link value
			if h {
				h = balanceLeft(pp)
			}
		}
		freeNode(q)    // return deleted node to pool
		removed = true // indicate that an item was removed
	}
	return value, removed, h
}
