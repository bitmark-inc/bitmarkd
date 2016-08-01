// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package avl

import ()

// find a specific item
func (tree *Tree) Search(key item) *node {
	return search(key, tree.root)
}

func search(key item, tree *node) *node {
	if nil == tree {
		return nil
	}

	switch tree.key.Compare(key) {
	case +1: // tree.key > key
		return search(key, tree.left)
	case -1: // tree.key < key
		return search(key, tree.right)
	default:
		return tree
	}
}
