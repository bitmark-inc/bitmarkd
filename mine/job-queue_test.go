// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package mine

import (
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/mode"
	"testing"
	"time"
)

// setup for tests
func setup() {
	mode.Initialise()
	mode.Set(mode.Normal)
}

// test job id conversion
func TestJobId(t *testing.T) {
	setup()

	tests := []struct {
		in       string
		expected jobIdentifier
	}{
		{"", 0},
		{"x", 0},
		{"12345", 0},
		{"1234", 0x1234},
		{"c0de", 0xc0de},
	}

	for i, test := range tests {

		jobId := stringToJobId(test.in)
		if test.expected != jobId {
			t.Errorf("%d: convert: %q  actual:  %s  expected: %s", i, test.in, jobId, test.expected)
		}
	}
}

// to test queue
type testItem struct {
	ids       []block.Digest
	addresses []block.MinerAddress
}

// test add/get
func TestJobQueue(t *testing.T) {
	setup()

	items := []testItem{
		{
			[]block.Digest{
				block.NewDigest([]byte("1234567890")),
				block.NewDigest([]byte("abcdefg")),
				block.NewDigest([]byte("ABCDEFG")),
			},
			[]block.MinerAddress{
				{Currency: "c11", Address: "a11"},
			},
		},
		{
			[]block.Digest{
				block.NewDigest([]byte("2143658709")),
				block.NewDigest([]byte("bacedgf")),
				block.NewDigest([]byte("ACBEDGF")),
			},
			[]block.MinerAddress{
				{Currency: "c21", Address: "a21"},
			},
		},
		{
			[]block.Digest{
				block.NewDigest([]byte("87894309594295240958")),
				block.NewDigest([]byte("qerwtrtrywt")),
				block.NewDigest([]byte("LHTJBKJBJGHV")),
			},
			[]block.MinerAddress{
				{Currency: "c31", Address: "a31"},
			},
		},
	}

	minTrees := make([][]block.Digest, len(items))
	for i, item := range items {
		minTrees[i] = block.MinimumMerkleTree(item.ids)
		if i > 0 && minTrees[i][0] == minTrees[i-1][0] {
			t.Fatalf("merkle trees conflict")
		}
	}

	initialiseJobQueue()

	simulatedId := jobIdentifier(0)
	simAlloc := func() jobIdentifier {
		simulatedId += 1
		return simulatedId
	}

	// check various clear/confirm
	for i := 1; i <= 4; i += 1 {

		// check initially empty
		checkClear(t)

		//timestamp for add
		timestamp := time.Now().UTC()

		// add some items
		jobQueue.add(items[0].ids, items[0].addresses, timestamp) // 0x0001
		simId1 := simAlloc()

		jobQueue.add(items[1].ids, items[1].addresses, timestamp) // 0x0002 - overwrites 0x0001
		simId2 := simAlloc()

		// ensure that second item is top
		checkTop(t, simId2, 1, true, items, minTrees)
		checkTop(t, simId2, 1, true, items, minTrees)

		// not able to confirm 0x0001
		checkNoConfirm(t, simId1)

		// add item again
		jobQueue.add(items[0].ids, items[0].addresses, timestamp)
		simId3 := simAlloc()

		// it should now be new top
		checkTop(t, simId3, 0, false, items, minTrees)

		// anothe item again
		jobQueue.add(items[2].ids, items[2].addresses, timestamp)
		simId4 := simAlloc()

		// it should now be new top
		checkTop(t, simId4, 2, false, items, minTrees)

		switch i {
		case 1:
			// empty the queue
			jobQueue.clear()
		case 2:
			checkConfirm(t, simId2)
		case 3:
			checkConfirm(t, simId3)
		case 4:
			checkConfirm(t, simId4)
		default:
			t.Fatalf("unhandled case: %d", i)
		}

		// check it is clear
		checkClear(t)
	}
}

func checkClear(t *testing.T) {
	jobId, merkle, addresses, timestamp, clean, ok := jobQueue.top()
	if ok {
		t.Fatalf("queue was not empty: %s %#v %#v %#v %v", jobId, merkle, addresses, timestamp, clean)
		return
	}

	// check it is clear
	if !jobQueue.isClear() {
		t.Fatal("queue was empty but not clear")
	}
}

func checkTop(t *testing.T, id jobIdentifier, index int, cleaned bool, items []testItem, minTrees [][]block.Digest) {

	// check that to was given a new id
	jobId, merkle, addresses, timestamp, clean, ok := jobQueue.top()
	if !ok {
		t.Fatal("queue was empty")
	}
	if id != jobId || !checkMerkle(t, minTrees[index], merkle) || cleaned != clean || !checkAddresses(t, items[index].addresses, addresses) {
		t.Errorf("queue top: id(%s) %#v %#v t(%#v) clean(%v)", jobId, merkle, addresses, timestamp, clean)
		if id != jobId {
			t.Errorf("  id expected: %s   actual: %s", id, jobId)
		}
		if cleaned != clean {
			t.Errorf("  clean expected: %v   actual: %v", cleaned, clean)
		}
	}

}

func checkAddresses(t *testing.T, addresses1 []block.MinerAddress, addresses2 []block.MinerAddress) bool {

	if len(addresses1) != len(addresses2) {
		t.Errorf("mismatched address lengths: %d to: %d", len(addresses1), len(addresses2))
		return false
	}

	result := true
	for i, a := range addresses1 {
		b := addresses2[i]
		if a.Currency != b.Currency || a.Address != b.Address {
			t.Errorf("address[%d] expected: (%q,%q) != actual: (%q,%q)", i, a.Currency, a.Address, b.Currency, b.Address)

			result = false
		}
	}

	return result
}

func checkMerkle(t *testing.T, minTree []block.Digest, merkle []block.Digest) bool {

	if minTree[0] != merkle[0] {
		t.Errorf("mismatched merkle root: %#v to: %#v", minTree[0], merkle[0])
		return false
	}
	return true
}

func checkNoConfirm(t *testing.T, id jobIdentifier) {
	if nil != jobQueue.confirm(id) {
		t.Fatalf("should not be able to confim overwritten id: %s", id)
	}
}

func checkConfirm(t *testing.T, id jobIdentifier) {
	if nil == jobQueue.confirm(id) {
		t.Fatalf("was not able to confim overwritten id: %s", id)
	}
}
