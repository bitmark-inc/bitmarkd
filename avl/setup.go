// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package avl

// type to hold the root node of a tree
type Tree struct {
	root  *Node
	count int
}

// create an initially empty tree
func New() *Tree {
	return &Tree{
		root:  nil,
		count: 0,
	}
}

// true if tree contains some data
func (tree *Tree) IsEmpty() bool {
	return nil == tree.root
}

// number of nodes currently in the tree
func (tree *Tree) Count() int {
	return tree.count
}

// read the key from a node
func (p *Node) Key() item {
	return p.key
}

// read the value from a node
func (p *Node) Value() interface{} {
	return p.value
}
