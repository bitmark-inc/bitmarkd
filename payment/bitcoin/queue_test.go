// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package bitcoin

import (
	// "bytes"
	"container/heap"
	// "encoding/binary"
	//"github.com/bitmark-inc/bitmarkd/payment/bitcoin"
	//"github.com/bitmark-inc/bitmarkd/account"
	//"github.com/bitmark-inc/bitmarkd/currency"
	// "github.com/bitmark-inc/bitmarkd/counter"
	// "github.com/bitmark-inc/bitmarkd/difficulty"
	// "github.com/bitmark-inc/bitmarkd/fault"
	// "github.com/bitmark-inc/bitmarkd/pool"
	//"github.com/bitmark-inc/bitmarkd/transactionrecord"
	//"github.com/bitmark-inc/logger"
	//"sync"
	"testing"
)

func TestQueue(t *testing.T) {

	items := []*priorityItem{
		&priorityItem{
			payId:         PayId{1, 2, 3, 4, 5, 6, 7, 8, 9, 0},
			txId:          "123456",
			confirmations: 14,
			blockNumber:   7,
		},
		&priorityItem{
			payId:         PayId{1, 2, 3, 4, 5, 6, 7, 8, 9, 6},
			txId:          "12342e3",
			confirmations: 24,
			blockNumber:   6,
		},
		&priorityItem{
			payId:         PayId{1, 2, 3, 4, 5, 6, 7, 8, 9, 7},
			txId:          "1234454",
			confirmations: 1,
			blockNumber:   67,
		},
		&priorityItem{
			payId:         PayId{1, 2, 3, 4, 5, 6, 7, 8, 9, 3},
			txId:          "1234u88765",
			confirmations: 3,
			blockNumber:   12,
		},
		&priorityItem{
			payId:         PayId{1, 2, 3, 4, 5, 6, 7, 8, 9, 2},
			txId:          "1234999",
			confirmations: 8,
			blockNumber:   146,
		},
		&priorityItem{
			payId:         PayId{1, 2, 3, 4, 5, 6, 7, 8, 9, 2},
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
