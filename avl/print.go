// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package avl

import (
	"fmt"
)

// to control the print routine
type branch int

const (
	root  branch = iota
	left  branch = iota
	right branch = iota
)

// Print - display an ASCII graphic representation of the tree
func (tree *Tree) Print(printData bool) int {
	return printTree(tree.root, "", root, printData)
}

// internal print - returns the maximum depth of the tree
func printTree(tree *Node, prefix string, br branch, printData bool) int {
	if tree == nil {
		return 0
	}
	rd := 0
	ld := 0
	if tree.right != nil {
		t := "       "
		if left == br {
			t = "|      "
		}
		rd = printTree(tree.right, prefix+t, right, printData)
	}
	switch br {
	case root:
		fmt.Printf("%s|------+ ", prefix)
	case left:
		fmt.Printf("%s\\------+ ", prefix)
	case right:
		fmt.Printf("%s/------+ ", prefix)
	}
	up := interface{}(nil)
	if tree.up != nil {
		up = tree.up.key
	}
	if printData {
		fmt.Printf("%q → %q ^%v %+2d/[%d,%d]\n", tree.key, tree.value, up, tree.balance, tree.leftNodes, tree.rightNodes)
	} else {
		fmt.Printf("%q ^%v\n", tree.key, up)
	}
	if tree.left != nil {
		t := "       "
		if right == br {
			t = "|      "
		}
		ld = printTree(tree.left, prefix+t, left, printData)
	}
	if rd > ld {
		return 1 + rd
	} else {
		return 1 + ld
	}
}
