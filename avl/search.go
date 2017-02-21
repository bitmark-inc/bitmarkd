// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package avl

// find a specific item
func (tree *Tree) Search(key item) *Node {
	return search(key, tree.root)
}

func search(key item, tree *Node) *Node {
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
