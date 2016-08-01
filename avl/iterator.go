// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package avl

import ()

// return the node with the lowest key value
func (tree *Tree) First() *node {
	return tree.root.first()
}

// internal: lowest node in a sub-tree
func (tree *node) first() *node {
	if nil == tree {
		return nil
	}
	for nil != tree.left {
		tree = tree.left
	}
	return tree
}

// return the node with the highest key value
func (tree *Tree) Last() *node {
	return tree.root.last()
}

// internal: highest node in a sub-tree
func (tree *node) last() *node {
	if nil == tree {
		return nil
	}
	for nil != tree.right {
		tree = tree.right
	}
	return tree
}

// given a node, return the node with the next highest key value or
// nil if no more nodes.
func (tree *node) Next() *node {
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

// given a node, return the node with the lowest key value or nil if
// no more nodes
func (tree *node) Prev() *node {
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
