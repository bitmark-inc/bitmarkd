// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package announce_test

import (
	"encoding/json"
	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/gnomon"
	"github.com/bitmark-inc/bitmarkd/pool"
	"github.com/bitmark-inc/bitmarkd/util"
	"os"
	"testing"
)

const (
	databaseFileName = "neighbours.leveldb"
)

func removeFiles() {
	os.RemoveAll(databaseFileName)
}

func setup(t *testing.T) {
	removeFiles()
	pool.Initialise(databaseFileName)
	announce.Initialise()
}

func teardown(t *testing.T) {
	announce.Finalise()
	pool.Finalise()
	removeFiles()
}

func add(t *testing.T, address string, newAddition bool) {
	f := util.Fingerprint([]byte(address)) // fake fingerprint
	p := announce.PeerData{
		// Type:        announce.TypePeer,
		// State:       announce.StateAllowed,
		Fingerprint: &f,
	}
	justAdded, err := announce.AddPeer(address, announce.TypePeer, &p)
	if nil != err {
		t.Errorf("Error on AddPeer: %v", err)
	}
	if justAdded != newAddition {
		t.Errorf("AddPeer: %s returned: %v  expected %v", address, justAdded, newAddition)
	}
}

func TestAnnounce(t *testing.T) {
	setup(t)
	defer teardown(t)

	add(t, "[::1]:2001", true)
	add(t, "127.0.0.1:1200", true)
	add(t, "[2404:6800:4008:c02::65]:443", true)
	add(t, "74.125.203.101:80", true)
	add(t, "127.0.0.1:1234", true)
	add(t, "127.0.0.1:1200", false)
	add(t, "127.0.0.1:2468", true)
	add(t, "[::1]:2001", false)

	check := [...]string{
		"[2404:6800:4008:c02::65]:443",
		"74.125.203.101:80",
		"127.0.0.1:1234",
	}

	start := &gnomon.Cursor{}
	recent, nextStart, err := announce.RecentPeers(start, len(check), announce.TypePeer)
	if nil != err {
		t.Errorf("Error on recent peers: %v", err)
		return
	}

	if start == nextStart {
		t.Errorf("no more data available, nextStart = %v", nextStart)
	}

	if len(recent) != len(check) {
		t.Errorf("got: %d  expected: %d", len(recent), len(check))
	}

	for i, ri := range recent {
		r := ri.(announce.RecentData)
		if i >= len(check) {
			t.Errorf("%d: Excess, got: '%s'  expected: Nothing", i, r.Address)
		} else if check[i] != r.Address {
			t.Errorf("%d: Mismatch, got: '%s'  expected: '%s'", i, r.Address, check[i])
		}
	}
}

// JSON test
func TestCursor(t *testing.T) {

	cursor := &gnomon.Cursor{}

	b, err := json.Marshal(cursor)
	if err != nil {
		t.Errorf("Error on json.Marshal: %v", err)
		return
	}

	expectedB := "\"000000000000000000000000\""
	if expectedB != string(b) {
		t.Errorf("json.Marshal returned: %s expected: %s", b, expectedB)
	}

	in := []byte("\"0012345678abcdef00002468\"")
	err = json.Unmarshal(in, &cursor)
	if err != nil {
		t.Errorf("Error on json.Unmarshal: %v", err)
		return
	}

	actualC, err := json.Marshal(cursor)
	if err != nil {
		t.Errorf("Error on json.Marshal: %v", err)
		return
	}

	expectedC := string(in)
	if string(actualC) != expectedC {
		t.Errorf("json.Unmarshal returned: %s expected: %s", actualC, expectedC)
	}

	null := []byte("null")
	err = json.Unmarshal(null, &cursor)
	if err != nil {
		t.Errorf("Error on json.Unmarshal: %v", err)
		return
	}

	actualC, err = json.Marshal(cursor)
	if err != nil {
		t.Errorf("Error on json.Marshal: %v", err)
		return
	}

	expectedC = string(null)
	if string(actualC) != expectedC {
		t.Errorf("json.Unmarshal returned: %s expected: %s", actualC, expectedC)
	}
}
