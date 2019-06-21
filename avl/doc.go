// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// Package avl - an AVL balanced tree with the addition of parent
// pointers to allow iteration through the nodes
//
// Note: an individual tree is not thread safe, so either access only
//       in a single go routine or use mutex/rwmutex to restrict
//       access.
//
// The base algorithm was described in an old book by Niklaus Wirth
// called Algorithms + Data Structures = Programs.
//
// This version allows for data associated with key, which can be
// overwritten by an insert with the same key.  Also delete no does
// not copy data around so that previous nodes can be deleted during
// iteration.
package avl
