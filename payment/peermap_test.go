// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package payment

import (
	"testing"

	"github.com/btcsuite/btcd/peer"
)

func TestPeerMapLen(t *testing.T) {
	address := "127.0.0.0:12345"

	m := NewPeerMap()

	if len(m.peers) != m.Len() {
		t.Fatalf("unexpected length of peer map: %d  actual: %d", len(m.peers), m.Len())
	}

	p, err := peer.NewOutboundPeer(&peer.Config{}, address)
	if err != nil {
		t.Fatalf("can not create peer: %s", err)
	}

	m.Add(address, p)

	if len(m.peers) != m.Len() {
		t.Fatalf("unexpected length of peer map: %d  actual: %d", len(m.peers), m.Len())
	}
}

func TestPeerMapAdd(t *testing.T) {
	address := "127.0.0.0:12345"

	m := NewPeerMap()

	p, err := peer.NewOutboundPeer(&peer.Config{}, address)
	if err != nil {
		t.Fatalf("can not create peer: %s", err)
	}

	m.Add(address, p)

	if len(m.peers) != 1 {
		t.Fatalf("unexpected length of peer map: %d  actual: %d", 1, m.Len())
	}
}

func TestPeerMapAddAndGet(t *testing.T) {
	address := "127.0.0.0:12345"

	m := NewPeerMap()

	p, err := peer.NewOutboundPeer(&peer.Config{}, address)
	if err != nil {
		t.Fatalf("can not create peer: %s", err)
	}
	m.Add(address, p)

	pActual := m.Get(address)

	if p.Addr() != pActual.Addr() {
		t.Fatalf("unexpected value of peer address: %s  actual: %s", p.Addr(), pActual.Addr())
	}

	pExist := m.Exist(address)

	if !pExist {
		t.Fatalf("expect true for an existing peer")
	}
}

func TestPeerMapNonExist(t *testing.T) {
	address := "127.0.0.0:12345"

	m := NewPeerMap()

	p := m.Get(address)

	if p != nil {
		t.Fatalf("should return nil for non-existing peer")
	}

	exist := m.Exist(address)
	if exist {
		t.Fatalf("expect false for a non-existing peer")
	}
}

func TestPeerMapAddAndDelete(t *testing.T) {
	address := "127.0.0.0:12345"

	m := NewPeerMap()

	p, err := peer.NewOutboundPeer(&peer.Config{}, address)
	if err != nil {
		t.Fatalf("can not create peer: %s", err)
	}

	m.Add(address, p)

	if m.Len() != 1 {
		t.Fatalf("unexpected length of peer map: %d  actual: %d", 1, m.Len())
	}

	m.Delete(address)

	if m.Len() != 0 {
		t.Fatalf("unexpected length of peer map: %d  actual: %d", 0, m.Len())
	}
}

func TestPeerMapDeleteNonExist(t *testing.T) {
	address1 := "127.0.0.0:12345"
	address2 := "127.0.0.0:12346"

	m := NewPeerMap()
	p, err := peer.NewOutboundPeer(&peer.Config{}, address1)
	if err != nil {
		t.Fatalf("can not create peer: %s", err)
	}

	m.Add(address1, p)

	m.Delete(address2)

	if m.Len() != 1 {
		t.Fatalf("unexpected length of peer map: %d  actual: %d", 1, m.Len())
	}
}
