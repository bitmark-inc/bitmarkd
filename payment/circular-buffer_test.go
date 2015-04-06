// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package payment

import (
	"github.com/bitmark-inc/bitmarkd/block"
	"testing"
)

// size for testing
const cbTestSize = 5

// helper
func cbAddOne(t *testing.T, c *circular, m block.MinerAddress) {
	c.put([]block.MinerAddress{m})
}

// check the address conversion to string
func TestAddressToString(t *testing.T) {
	// bitcoin addresses
	items := []struct {
		a block.MinerAddress
		s string
	}{
		{
			s: "\x07bitcoin\x22n1CuYF7iKoAxUicVT2CmyJTQdXTtMWTPNd",
			a: block.MinerAddress{Currency: "bitcoin", Address: "n1CuYF7iKoAxUicVT2CmyJTQdXTtMWTPNd"},
		},
		{
			s: "\x07bitcoin\x22mgnZvJCMtSjaf9AEG7nd8hLtsVis5QfAMp",
			a: block.MinerAddress{Currency: "bitcoin", Address: "mgnZvJCMtSjaf9AEG7nd8hLtsVis5QfAMp"},
		},
		{
			s: "\x0bjusttesting\x0e!@#$%^&*()_+{}",
			a: block.MinerAddress{Currency: "justtesting", Address: "!@#$%^&*()_+{}"},
		},
	}

	// test all
	for i, item := range items {
		actual := item.a.String()
		if actual != item.s {
			t.Errorf("%d: convert: %q  got: %q  expected %q", i, item.a, actual, item.s)
		}
	}
}

// struct for test data
type testItem struct {
	p []bool
	a []block.MinerAddress
}

// main circular buffer test
func TestCircularBuffer(t *testing.T) {

	p := newCircular(cbTestSize)

	// bitcoin addresses
	bitcoinAddress := []block.MinerAddress{
		{Currency: "bitcoin", Address: "n1CuYF7iKoAxUicVT2CmyJTQdXTtMWTPNd"},
		{Currency: "bitcoin", Address: "mx8hBVZdpYGZW7avdjw1vMeR4T1ALehxah"},
		{Currency: "bitcoin", Address: "mojV5g3fnYDU6Z7vfktZ9AH5fNyZppvonQ"},
		{Currency: "bitcoin", Address: "mgnZvJCMtSjaf9AEG7nd8hLtsVis5QfAMp"},
		{Currency: "bitcoin", Address: "mjLAk8viCmnCHp7CEf7h6xWtrBewtRaM83"},
		{Currency: "bitcoin", Address: "mq625RYFA9HKFjyeVWQhGJ8QS59tx3bQGp"},
		{Currency: "bitcoin", Address: "mstyrWVLYYV2agrADWk3B89q9cPtY5qQ8w"},
		{Currency: "bitcoin", Address: "muvE7tkTXfHukRv1V3i4UDJp49zws5sotB"},
		{Currency: "bitcoin", Address: "mrw8uBr86RFjMs7LgijrutNYun4ySkohXv"},
		{Currency: "bitcoin", Address: "mnLUa4JHV2UF1AJJNEZPj9T2gpYHJ1jg8s"},
	}

	// some data items
	testItems := []testItem{
		{p: []bool{false, true, true},
			a: []block.MinerAddress{
				bitcoinAddress[0],
			}},
		{p: []bool{true, false, false},
			a: []block.MinerAddress{
				bitcoinAddress[1],
			}},
		{p: []bool{false, false, false},
			a: []block.MinerAddress{
				bitcoinAddress[2],
			}},
		{p: []bool{true, true, true},
			a: []block.MinerAddress{
				bitcoinAddress[3],
			}},
		{p: []bool{false, false, false},
			a: []block.MinerAddress{
				bitcoinAddress[4],
			}},
		{p: []bool{true, false, false},
			a: []block.MinerAddress{
				bitcoinAddress[5],
			}},
		{p: []bool{true, true, false},
			a: []block.MinerAddress{
				bitcoinAddress[6],
			}},
		{p: []bool{true, true, true},
			a: []block.MinerAddress{
				bitcoinAddress[7],
			}},
		{p: []bool{false, false, true},
			a: []block.MinerAddress{
				bitcoinAddress[8],
			}},
		{p: []bool{false, false, false},
			a: []block.MinerAddress{
				bitcoinAddress[9],
			}},
	}

	// add more items than cacheSize
	cbAddOne(t, p, bitcoinAddress[0])
	cbAddOne(t, p, bitcoinAddress[1])
	cbAddOne(t, p, bitcoinAddress[2])
	cbAddOne(t, p, bitcoinAddress[3])
	cbAddOne(t, p, bitcoinAddress[1])
	cbAddOne(t, p, bitcoinAddress[4])
	cbAddOne(t, p, bitcoinAddress[1])
	cbAddOne(t, p, bitcoinAddress[5])
	cbAddOne(t, p, bitcoinAddress[6])
	cbAddOne(t, p, bitcoinAddress[3])
	cbAddOne(t, p, bitcoinAddress[7])

	pass := 0
	checkPresence(t, p, testItems, &pass)

	// modify cache and recheck
	cbAddOne(t, p, bitcoinAddress[0])
	cbAddOne(t, p, bitcoinAddress[0])

	checkPresence(t, p, testItems, &pass)

	cbAddOne(t, p, bitcoinAddress[8])

	checkPresence(t, p, testItems, &pass)
}

// check presence of all items
func checkPresence(t *testing.T, p *circular, testItems []testItem, passPtr *int) {

	pass := *passPtr
	*passPtr = pass + 1

	for i, item := range testItems {
		for j, a := range item.a {
			present := p.isPresent(a)
			if present != item.p[pass] {
				t.Errorf("P%d: %d: presence error: a[%d] = %q  got: %v  expected %v", pass+1, i, j, a, present, item.p[pass])
			}
		}
	}

}
