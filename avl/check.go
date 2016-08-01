// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package avl

import (
	"fmt"
)

// check the up pointers for consistency
func (tree *Tree) CheckUp() bool {
	return checkup(tree.root, nil)
}

// internal: consistency checker
func checkup(p *node, up *node) bool {
	if nil == p {
		return true
	}
	if p.up != up {
		fmt.Printf("fail at node: %v   actual: %v  expected: %v\n", p.key, p.up.key, up.key)
		return false
	}
	if !checkup(p.left, p) {
		return false
	}
	return checkup(p.right, p)
}
