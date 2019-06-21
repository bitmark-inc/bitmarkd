// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package avl

// First - return the node with the lowest key value
func (tree *Tree) First() *Node {
	return tree.root.first()
}

// internal: lowest node in a sub-tree
func (tree *Node) first() *Node {
	if nil == tree {
		return nil
	}
	for nil != tree.left {
		tree = tree.left
	}
	return tree
}

// Last - return the node with the highest key value
func (tree *Tree) Last() *Node {
	return tree.root.last()
}

// internal: highest node in a sub-tree
func (tree *Node) last() *Node {
	if nil == tree {
		return nil
	}
	for nil != tree.right {
		tree = tree.right
	}
	return tree
}

// Next - given a node, return the node with the next highest key
// value or nil if no more nodes.
func (tree *Node) Next() *Node {
	if nil == tree.right {
		key := tree.key
		for {
			tree = tree.up
			if nil == tree {
				return nil
			}
			if 1 == tree.key.Compare(key) { // tree.key > key
				return tree
			}
		}
	}
	return tree.right.first()
}

// Prev - given a node, return the node with the lowest key value or
// nil if no more nodes
func (tree *Node) Prev() *Node {
	if nil == tree.left {
		key := tree.key
		for {
			tree = tree.up
			if nil == tree {
				return nil
			}
			if -1 == tree.key.Compare(key) { // tree.key < key
				return tree
			}
		}
	}
	return tree.left.last()
}
