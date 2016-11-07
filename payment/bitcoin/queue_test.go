// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package bitcoin

import (
	"container/heap"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"testing"
)

func stringToPayId(t *testing.T, s string) (payId reservoir.PayId) {
	s = s + "dd4b3c88a71bf2c7c29df2e39b5ad48e6af09e5075295dba73838a2ee28dcbfd" + s
	err := payId.UnmarshalText([]byte(s))
	if nil != err {
		t.Fatalf("converting: %q to pay id error: %s", s, err)
	}
	return
}

func TestQueue(t *testing.T) {

	items := []*priorityItem{
		{
			payId:         stringToPayId(t, "12345567edac76f0"),
			txId:          "123456",
			confirmations: 14,
			blockNumber:   7,
		},
		{
			payId:         stringToPayId(t, "12345567edac76fb"),
			txId:          "12342e3",
			confirmations: 24,
			blockNumber:   6,
		},
		{
			payId:         stringToPayId(t, "12345567edac76f9"),
			txId:          "1234454",
			confirmations: 1,
			blockNumber:   67,
		},
		{
			payId:         stringToPayId(t, "12345567edac76fe"),
			txId:          "1234u88765",
			confirmations: 3,
			blockNumber:   12,
		},
		{
			payId:         stringToPayId(t, "12345567edac76f1"),
			txId:          "1234999",
			confirmations: 8,
			blockNumber:   146,
		},
		{
			payId:         stringToPayId(t, "12345567edac76f5"),
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
		t.Logf("item: %#v", item)
		actual = append(actual, item.blockNumber)
	}

	for i, ex := range expected {
		if ex != actual[i] {
			t.Errorf("item: %d  expected: %d  actual: %d", i, ex, actual[i])
		}
	}
}
