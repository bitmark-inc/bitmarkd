// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package receptor

import (
	"bytes"
	"crypto/rand"
	"math/big"
	"sync"
	"time"

	"github.com/bitmark-inc/bitmarkd/messagebus"

	"github.com/bitmark-inc/bitmarkd/fault"

	"github.com/bitmark-inc/bitmarkd/announce/id"

	"github.com/bitmark-inc/bitmarkd/announce/helper"
	"github.com/bitmark-inc/bitmarkd/announce/parameter"

	"github.com/bitmark-inc/logger"

	"github.com/bitmark-inc/bitmarkd/avl"
)

// format for timestamps
const timeFormat = "2006-01-02 15:04:05"

// Receptor - interface for receptor operations
type Receptor interface {
	Add([]byte, []byte, uint64) bool
	SetSelf([]byte, []byte) error
	Next([]byte) ([]byte, []byte, time.Time, error)
	Random([]byte) ([]byte, []byte, time.Time, error)
	ReBalance()
	UpdateTime([]byte, time.Time)
	IsChanged() bool
	Change(bool)
	IsInitialised() bool
	Connectable() *avl.Tree
	ID() id.ID
	Self() *avl.Node
	SelfListener() []byte
	Expire()
}

type receptor struct {
	sync.RWMutex
	connectable *avl.Tree
	self        *avl.Node
	changed     bool
	id          id.ID
	log         *logger.L
	initialised bool
	listeners   []byte
}

// Add - add a connectable entity to in-memory tree
// returns:
//   true  if this was a new/updated entry
//   false if the update was within the limits (to prevent continuous relaying)
func (r *receptor) Add(publicKey []byte, listeners []byte, timestamp uint64) bool {
	r.Lock()
	defer r.Unlock()

	ts := helper.ResetFutureTimestampToNow(timestamp)
	if helper.IsExpiredAfterDuration(ts, parameter.ExpiryInterval) {
		return false
	}

	e := &Entity{
		PublicKey: publicKey,
		Listeners: listeners,
		Timestamp: ts,
	}

	if node, _ := r.connectable.Search(id.ID(publicKey)); nil != node {
		e := node.Value().(*Entity)

		if ts.Sub(e.Timestamp) < parameter.RebroadcastInterval {
			return false
		}
	}

	// add or update the timestamp in the tree
	recordAdded := r.connectable.Insert(id.ID(publicKey), e)

	r.log.Debugf("added: %t  nodes in the connectable tree: %d", recordAdded, r.connectable.Count())

	// if adding this nodes data
	if bytes.Equal(r.id, publicKey) {
		return false
	}

	if recordAdded {
		r.changed = true
	}

	return true
}

// SetSelf - called by announce initialisation to setup own announcement data
func (r *receptor) SetSelf(publicKey []byte, listeners []byte) error {
	r.Lock()

	if r.initialised {
		r.Unlock()
		return fault.AlreadyInitialised
	}
	r.id = publicKey
	r.listeners = listeners
	r.initialised = true
	r.Unlock()

	r.Add(publicKey, listeners, uint64(time.Now().Unix()))

	r.Lock()
	r.self, _ = r.connectable.Search(id.ID(publicKey))
	r.Unlock()

	r.ReBalance()

	return nil
}

// ReBalance - re-balance tree for better connections
func (r *receptor) ReBalance() {
	r.Lock()
	defer r.Unlock()

	log := r.log

	if nil == r.self {
		log.Errorf("determineConnections called to early")
		return // called to early
	}

	// locate this node in the tree
	_, index := r.connectable.Search(r.self.Key())
	count := r.connectable.Count()
	log.Infof("N0: index: %d  tree: %d  public key: %x", index, count, r.id)

	// various increment values
	e := count / 8
	q := count / 4
	h := count / 2

	jump := 3      // to deal with N3/P3 and too few nodes
	if count < 4 { // if insufficient
		jump = 1 // just duplicate N1/P1
	}

	names := [11]string{
		"N1",
		"N3",
		"X1",
		"X2",
		"X3",
		"X4",
		"X5",
		"X6",
		"X7",
		"P1",
		"P3",
	}

	// compute all possible offsets
	// if count is too small then there will be duplicate offsets
	var n [11]int
	n[0] = index + 1             // N1 (+1)
	n[1] = index + jump          // N3 (+3)
	n[2] = e + index             // X⅛
	n[3] = q + index             // X¼
	n[4] = q + e + index         // X⅜
	n[5] = h + index             // X½
	n[6] = h + e + index         // X⅝
	n[7] = h + q + index         // X¾
	n[8] = h + q + e + index     // X⅞
	n[9] = index + count - 1     // P1 (-1)
	n[10] = index + count - jump // P3 (-3)

	u := -1
deduplicate:
	for i, v := range n {
		if v == index || v == u {
			continue deduplicate
		}
		u = v
		if v >= count {
			v -= count
		}
		node := r.connectable.Get(v)
		if nil != node {
			e := node.Value().(*Entity)
			if nil != e {
				log.Infof("%s: connectable entity: %s", names[i], e)
				messagebus.Bus.Connector.Send(names[i], e.PublicKey, e.Listeners)
			}
		}
	}
}

// Next - fetch data for next node in the ring for a given public key
func (r *receptor) Next(publicKey []byte) ([]byte, []byte, time.Time, error) {
	r.Lock()
	defer r.Unlock()

	node, _ := r.connectable.Search(id.ID(publicKey))
	if nil != node {
		node = node.Next()
	}
	if nil == node {
		node = r.connectable.First()
	}
	if nil == node {
		return nil, nil, time.Now(), fault.InvalidPublicKey
	}
	e := node.Value().(*Entity)
	return e.PublicKey, e.Listeners, e.Timestamp, nil
}

// Random - fetch a random node data in the ring not matching a given public key
func (r *receptor) Random(publicKey []byte) ([]byte, []byte, time.Time, error) {
	r.Lock()
	defer r.Unlock()

loop:
	for tries := 1; tries <= 5; tries += 1 {
		max := big.NewInt(int64(r.connectable.Count()))
		num, err := rand.Int(rand.Reader, max)
		if nil != err {
			continue loop
		}

		n := int(num.Int64()) // 0 … max-1

		node := r.connectable.Get(n)
		if nil == node {
			node = r.connectable.First()
		}
		if nil == node {
			break loop
		}
		e := node.Value().(*Entity)
		if bytes.Equal(e.PublicKey, r.id) || bytes.Equal(e.PublicKey, publicKey) {
			continue loop
		}
		return e.PublicKey, e.Listeners, e.Timestamp, nil
	}
	return []byte{}, []byte{}, time.Now(), fault.InvalidPublicKey
}

// UpdateTime - initialised timestamp for connectable entity with given public key
func (r *receptor) UpdateTime(publicKey []byte, timestamp time.Time) {
	r.Lock()
	defer r.Unlock()

	node, _ := r.connectable.Search(id.ID(publicKey))
	log := r.log
	if nil == node {
		log.Errorf("The connectable entity with public key %x is not existing in tree", publicKey)
		return
	}

	e := node.Value().(*Entity)
	e.Timestamp = timestamp
}

// Change - update flag of changed status
func (r *receptor) Change(changed bool) {
	r.changed = changed
}

// IsChanged - return flag of changed status
func (r *receptor) IsChanged() bool {
	return r.changed
}

// IsInitialised - return flag of initialised status
func (r *receptor) IsInitialised() bool {
	return r.initialised
}

// Connectable - return tree of all connectable nodes
func (r *receptor) Connectable() *avl.Tree {
	return r.connectable
}

// ID - public key of a node
func (r *receptor) ID() id.ID {
	return r.id
}

// Self - return this node data
func (r *receptor) Self() *avl.Node {
	return r.self
}

// SelfListener - return self listener
func (r *receptor) SelfListener() []byte {
	return r.listeners
}

// Expire - remove outdated node
func (r *receptor) Expire() {
	r.Lock()
	defer r.Unlock()

	now := time.Now()
	nextNode := r.connectable.First()
	log := r.log

scanNodes:
	for node := nextNode; nil != node; node = nextNode {

		peer := node.Value().(*Entity)
		key := node.Key()

		nextNode = node.Next()

		// skip this node's entry
		if bytes.Equal(r.id, peer.PublicKey) {
			continue scanNodes
		}
		log.Debugf("public key: %x timestamp: %s", peer.PublicKey, peer.Timestamp.Format(timeFormat))
		if peer.Timestamp.Add(parameter.ExpiryInterval).Before(now) {
			r.connectable.Delete(key)
			r.changed = true
			messagebus.Bus.Connector.Send("@D", peer.PublicKey, peer.Listeners) //@D means: @->Internal Command, D->delete
			log.Infof("Peer Expired! public key: %x timestamp: %s is removed", peer.PublicKey, peer.Timestamp.Format(timeFormat))
		}
	}
}

// New - return Receptor interface
func New(log *logger.L) Receptor {
	return &receptor{
		connectable: avl.New(),
		log:         log,
	}
}
