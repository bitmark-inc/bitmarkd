// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package avl

// Get - access specific item by index
func (tree *Tree) Get(index int) *Node {
	if index < 0 || index >= tree.Count() {
		return nil
	}
	return get(index, tree.root)
}

func get(index int, tree *Node) *Node {
	if nil == tree {
		return nil
	}

	nl := tree.leftNodes

	if index < nl {
		return get(index, tree.left)
	}
	if index > nl {
		// subtract left nodes + 1 (for this node)
		return get(index-nl-1, tree.right)
	}
	return tree
}
