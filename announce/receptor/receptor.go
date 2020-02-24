// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package receptor

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/gogo/protobuf/proto"
	p2pPeer "github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/bitmark-inc/bitmarkd/announce/helper"
	"github.com/bitmark-inc/bitmarkd/announce/id"
	"github.com/bitmark-inc/bitmarkd/announce/parameter"
	"github.com/bitmark-inc/bitmarkd/avl"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/logger"
)

const timeFormat = "2006-01-02 15:04:05"

type Receptor interface {
	Add(p2pPeer.ID, []ma.Multiaddr, uint64) bool
	Changed() bool
	Change(bool)
	IsSet() bool
	Next(p2pPeer.ID) (p2pPeer.ID, []ma.Multiaddr, time.Time, error)
	Random(p2pPeer.ID) (p2pPeer.ID, []ma.Multiaddr, time.Time, error)
	SetSelf(p2pPeer.ID, []ma.Multiaddr) error
	Self() *avl.Node
	SelfAddress() []ma.Multiaddr
	Tree() *avl.Tree
	ID() p2pPeer.ID
	BinaryID() []byte
	ShortID() string
	UpdateTime(p2pPeer.ID, time.Time)
	ReBalance()
	Expire()
}

type receptor struct {
	sync.RWMutex
	selfID    p2pPeer.ID
	tree      *avl.Tree
	self      *avl.Node
	changed   bool
	set       bool
	listeners []ma.Multiaddr
	log       *logger.L
}

func New(log *logger.L) Receptor {
	return &receptor{
		tree: avl.New(),
		log:  log,
	}
}

// UpdateTime - update time by id
func (r *receptor) UpdateTime(pID p2pPeer.ID, timestamp time.Time) {
	r.Lock()
	defer r.Unlock()

	node, _ := r.tree.Search(id.ID(pID))
	if nil == node {
		r.log.Errorf("Public key %x is not existing in tree", pID.Pretty())
		return
	}

	data := node.Value().(*Data)
	data.Timestamp = timestamp
}

func (r receptor) ShortID() string {
	return r.selfID.ShortString()
}

func (r receptor) BinaryID() []byte {
	myID, _ := r.selfID.MarshalBinary()
	return myID
}

func (r receptor) SelfAddress() []ma.Multiaddr {
	return r.listeners
}

func (r receptor) ID() p2pPeer.ID {
	return r.selfID
}

func (r receptor) IsSet() bool {
	return r.set
}

func (r receptor) Tree() *avl.Tree {
	return r.tree
}

func (r receptor) Self() *avl.Node {
	return r.self
}

func (r *receptor) SetSelf(peerID p2pPeer.ID, addrs []ma.Multiaddr) error {
	if r.set {
		return fault.AlreadyInitialised
	}
	r.selfID = peerID
	r.listeners = addrs
	r.set = true

	r.Add(peerID, addrs, uint64(time.Now().Unix()))
	r.self, _ = r.tree.Search(id.ID(peerID))
	r.ReBalance()

	return nil
}

func (r *receptor) Add(peerID p2pPeer.ID, listeners []ma.Multiaddr, timestamp uint64) bool {
	r.Lock()
	defer r.Unlock()

	ts := helper.ResetFutureTimeToNow(timestamp)
	if helper.IsExpiredAfterDuration(ts, parameter.ExpiryInterval) {
		return false
	}

	d := &Data{
		ID:        peerID,
		Listeners: listeners,
		Timestamp: ts,
	}
	// TODO: Take care of update and replace base on multi-address protocol
	if node, _ := r.tree.Search(id.ID(peerID)); nil != node {
		peer := node.Value().(*Data)

		if ts.Sub(peer.Timestamp) < parameter.RebroadcastInterval {
			return false
		}

	}

	// add or update the Timestamp in the tree
	recordAdded := r.tree.Insert(id.ID(peerID), d)

	r.log.Infof("Peer Added:  ID: %s,  add:%t  nodes in the r tree: %d", r.selfID.String(), recordAdded, r.tree.Count())

	// if adding this nodes data
	if util.IDEqual(r.selfID, peerID) {
		return false
	}

	if recordAdded {
		r.changed = true
	}

	return true
}

func (r receptor) Changed() bool {
	return r.changed
}

func (r *receptor) Change(b bool) {
	r.changed = b
}

func (r *receptor) Next(peerID p2pPeer.ID) (p2pPeer.ID, []ma.Multiaddr, time.Time, error) {
	r.Lock()
	defer r.Unlock()

	node, _ := r.tree.Search(id.ID(peerID))
	if nil != node {
		node = node.Next()
	}
	if nil == node {
		node = r.tree.First()
	}
	if nil == node {
		return p2pPeer.ID(""), nil, time.Now(), fault.InvalidPublicKey
	}
	peer := node.Value().(*Data)
	return peer.ID, peer.Listeners, peer.Timestamp, nil
}

func (r *receptor) Random(peerID p2pPeer.ID) (p2pPeer.ID, []ma.Multiaddr, time.Time, error) {
	r.Lock()
	defer r.Unlock()

loop:
	for tries := 1; tries <= 5; tries += 1 {
		max := big.NewInt(int64(r.tree.Count()))
		num, err := rand.Int(rand.Reader, max)
		if nil != err {
			continue loop
		}

		n := int(num.Int64()) // 0 … max-1

		node := r.tree.Get(n)
		if nil == node {
			node = r.tree.First()
		}
		if nil == node {
			break loop
		}
		peer := node.Value().(*Data)
		if util.IDEqual(peer.ID, r.selfID) || util.IDEqual(peer.ID, peerID) {
			continue loop
		}
		return peer.ID, peer.Listeners, peer.Timestamp, nil
	}
	return p2pPeer.ID(""), nil, time.Now(), fault.InvalidPublicKey
}

func (r *receptor) Expire() {
	now := time.Now()
	nextNode := r.tree.First()
loop:
	for node := nextNode; nil != node; node = nextNode {

		p := node.Value().(*Data)
		key := node.Key()

		nextNode = node.Next()

		// skip this node's entry
		if r.ID().String() == p.ID.String() {
			continue loop
		}
		if p.Timestamp.Add(parameter.ExpiryInterval).Before(now) {
			r.tree.Delete(key)
			r.Change(true)
			util.LogDebug(r.log, util.CoReset, fmt.Sprintf("expirePeer : ID: %v! Timestamp: %s", p.ID.ShortString(), p.Timestamp.Format(timeFormat)))
			idBinary, errID := p.ID.Marshal()
			if nil == errID {
				messagebus.Bus.P2P.Send("@D", idBinary)
				util.LogInfo(r.log, util.CoYellow, fmt.Sprintf("--><-- Send @D to P2P  ID: %v", p.ID.ShortString()))
			}
		}
	}
}

func (r *receptor) ReBalance() {
	if nil == r.self {
		util.LogWarn(r.log, util.CoRed, fmt.Sprintf("ReBalance called to early"))
		return // called to early
	}

	// locate this node in the tree
	_, index := r.tree.Search(r.self.Key())
	count := r.tree.Count()

	// various increment values
	e := count / 8
	q := count / 4
	h := count / 2

	jump := 3      // to deal with N3/P3 and too few nodes
	if count < 4 { // if insufficient
		jump = 1 // just duplicate N1/P1
	}

	// X1 - X7: each node is 12.5% ahead of current node on the tree
	// N1: next node
	// P1: previous node
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
		node := r.tree.Get(v)
		if nil != node {
			p := node.Value().(*Data)
			if nil != p {
				idBinary, errID := p.ID.Marshal()
				pbAddr := util.GetBytesFromMultiaddr(p.Listeners)
				pbAddrBinary, errMarshal := proto.Marshal(&Addrs{Address: pbAddr})
				if nil == errID && nil == errMarshal {
					messagebus.Bus.P2P.Send(names[i], idBinary, pbAddrBinary)
					util.LogDebug(r.log, util.CoYellow, fmt.Sprintf("--><-- determine send to P2P %v : %s  address: %x ", names[i], p.ID.ShortString(), printBinaryAddrs(pbAddrBinary)))
				}
			}

		}
	}
}

func printBinaryAddrs(addrs []byte) string {
	maAddrs := Addrs{}
	err := proto.Unmarshal(addrs, &maAddrs)
	if err != nil {
		return ""
	}
	printAddrs := util.PrintMaAddrs(util.GetMultiAddrsFromBytes(maAddrs.Address))
	return printAddrs
}

func AddrToString(addrs []byte) string {
	maAddrs := Addrs{}
	err := proto.Unmarshal(addrs, &maAddrs)
	if err != nil {
		return ""
	}

	str := util.PrintMaAddrs(util.GetMultiAddrsFromBytes(maAddrs.Address))
	return str
}
