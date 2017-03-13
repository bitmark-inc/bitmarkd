// Copyright (c) 2014-2017 Bitmark Inc.
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

// Root return the root node of the tree
func (tree *Tree) Root() *Node {
	return tree.root
}

// GetChildrenByDepth will returns all children in a specific depth of a tree
func (p *Node) GetChildrenByDepth(depth uint) []*Node {
	nodes := []*Node{}

	if depth == 0 {
		nodes = []*Node{p}
	} else {
		left := p.left
		right := p.right
		if left != nil {
			nodes = append(nodes, left.GetChildrenByDepth(depth-1)...)
		}

		if right != nil {
			nodes = append(nodes, right.GetChildrenByDepth(depth-1)...)
		}
	}
	return nodes
}

// read the key from a node
func (p *Node) Key() item {
	return p.key
}

// read the value from a node
func (p *Node) Value() interface{} {
	return p.value
}

// return parent node of a node
func (p *Node) Parent() *Node {
	return p.up
}

// get the depth of a node
func (p *Node) Depth() uint {
	count := uint(0)
	parent := p.up
	for parent != nil {
		count += 1
		parent = parent.up
	}
	return count
}
