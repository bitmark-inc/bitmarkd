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

// get the order of a node with a specific key in a tree
func (p *Node) GetOrder(key item) uint {
	iterNode := p.first()
	lastKey := p.last().Key()
	order := uint(0)

	for iterNode.Key().Compare(key) != 0 && iterNode.Key().Compare(lastKey) != 0 {
		order += 1
		iterNode = iterNode.Next()
	}

	return order
}

// get the node of a tree in order
func (p *Node) GetNodeByOrder(order uint) *Node {
	node := p.first()
	for i := uint(0); i < order; i++ {
		if node == p.last() {
			break
		}
		node = node.Next()
	}
	return node
}
