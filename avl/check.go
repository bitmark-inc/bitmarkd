// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package avl

import (
	"fmt"
)

// CheckUp - check the up pointers for consistency
func (tree *Tree) CheckUp() bool {
	return checkUp(tree.root, nil)
}

// internal: consistency checker
func checkUp(p *Node, up *Node) bool {
	if nil == p {
		return true
	}
	if p.up != up {
		fmt.Printf("fail at node: %v   actual: %v  expected: %v\n", p.key, p.up.key, up.key)
		return false
	}
	if !checkUp(p.left, p) {
		return false
	}
	return checkUp(p.right, p)
}

// CheckCounts - check left and right node counts are ok
func (tree *Tree) CheckCounts() bool {
	b, _ := checkCounts(tree.root)
	return b
}

func checkCounts(p *Node) (bool, int) {
	if nil == p {
		return true, 0
	}
	bl := true
	nl := 0
	if nil != p.left {
		bl, nl = checkCounts(p.left)
		if p.leftNodes != nl {
			fmt.Printf("fail at node: %v  left actual: %d  record: %d\n", p.key, nl, p.leftNodes)
			bl = false
		}
	}
	br := true
	nr := 0
	if nil != p.right {
		br, nr = checkCounts(p.right)
		if p.rightNodes != nr {
			fmt.Printf("fail at node: %v  right actual: %d  record: %d\n", p.key, nr, p.rightNodes)
			br = false
		}
	}
	return bl && br, 1 + nl + nr
}
