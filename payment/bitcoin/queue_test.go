// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package bitcoin

import (
	"container/heap"
	"testing"
)

func TestQueue(t *testing.T) {

	items := []*priorityItem{
		&priorityItem{
			payId:         "1234ac76f0",
			txId:          "123456",
			confirmations: 14,
			blockNumber:   7,
		},
		&priorityItem{
			payId:         "1234ac76fb",
			txId:          "12342e3",
			confirmations: 24,
			blockNumber:   6,
		},
		&priorityItem{
			payId:         "1234ac76f9",
			txId:          "1234454",
			confirmations: 1,
			blockNumber:   67,
		},
		&priorityItem{
			payId:         "1234ac76fe",
			txId:          "1234u88765",
			confirmations: 3,
			blockNumber:   12,
		},
		&priorityItem{
			payId:         "1234ac76f1",
			txId:          "1234999",
			confirmations: 8,
			blockNumber:   146,
		},
		&priorityItem{
			payId:         "1234ac76f5",
			txId:          "1234777",
			confirmations: 1,
			blockNumber:   46,
		},
	}

	// block numbers in ascending order
	expected := []uint64{6, 7, 12, 46, 67, 146}

	pq := new(priorityQueue)
	heap.Init(pq)
	for _, item := range items {
		heap.Push(pq, item)
	}

	actual := []uint64{}
	for pq.Len() > 0 {
		item := heap.Pop(pq).(*priorityItem)
		t.Logf("item: %v", item)
		actual = append(actual, item.blockNumber)
	}

	for i, ex := range expected {
		if ex != actual[i] {
			t.Errorf("item: %d  expected: %d  actual: %d", i, ex, actual[i])
		}
	}
}
