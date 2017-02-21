// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package merkle

import (
//"github.com/bitmark-inc/bitmarkd/merkledigest"
)

// compute minimum merkle root from a set of transaction ids
//
// structure is:
//   1. N * transaction digests
//   2. level 1..m digests
//   3. merkle root digest
func FullMerkleTree(txIds []Digest) []Digest {

	// compute length of ids + all tree levels including root
	idCount := len(txIds)

	totalLength := 1 // all ids + space for the final root
	for n := idCount; n > 1; n = (n + 1) / 2 {
		totalLength += n
	}

	// add initial ids
	tree := make([]Digest, totalLength)
	copy(tree[:], txIds)

	n := idCount
	j := 0
	for workLength := idCount; workLength > 1; workLength = (workLength + 1) / 2 {
		for i := 0; i < workLength; i += 2 {
			k := j + 1
			if i+1 == workLength {
				k = j // compensate for odd number
			}
			tree[n] = NewDigest(append(tree[j][:], tree[k][:]...))
			n += 1
			j = k + 1
		}
	}
	return tree
}

// merkle hashing
//
// build a minimised tree
func MinimumMerkleTree(ids []Digest) []Digest {

	length := len(ids)

	// ensure length is within 24 bit positive number
	if length <= 0 || length > 0x1000000 {
		return nil
	}

	// output length =  tx[0] + workspace = 1 + len/2
	outputLength := length/2 + 1

	tree := make([]Digest, outputLength)
	tree[0] = ids[0]

	// build minimum merkle - two sections 1.ids[]->tree[]; 2.tree[]->tree[]
	finish := length
	for start := 1; start < finish; start += 1 {
		n := start
		for i := start; i < finish; i += 2 {
			j := i + 1
			if j >= finish {
				j = i // compensate for odd number
			}
			var b []byte
			if 1 == start {
				b = append(ids[i][:], ids[j][:]...)
			} else {
				b = append(tree[i][:], tree[j][:]...)
			}
			tree[n] = NewDigest(b)
			n += 1
		}
		finish = n
	}
	return tree[:finish]
}
