// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package announce

import (
	"github.com/bitmark-inc/bitmarkd/avl"
	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/logger"
	"net"
	"strings"
	"sync"
)

// type of listener
const (
	TypeRPC  = iota
	TypePeer = iota
)

// globals for background proccess
type announcerData struct {
	sync.RWMutex // to allow locking

	// logger
	log *logger.L

	// this node's packed annoucements
	publicKey   []byte
	broadcasts  []byte
	listeners   []byte
	fingerprint [32]byte
	rpcs        []byte
	peerSet     bool
	rpcsSet     bool

	// trre of nodes available
	peerTree *avl.Tree
	thisNode *avl.Node // this node's position in the tree
	change   bool      // tree was changed

	n1 *avl.Node // first neighbour
	n3 *avl.Node // third neighbour

	ann announcer

	// for background
	background *background.T

	// set once during initialise
	initialised bool
}

// global data
var globalData announcerData

// initialise the announcement system
func Initialise() error {

	globalData.Lock()
	defer globalData.Unlock()

	// no need to start if already started
	if globalData.initialised {
		return fault.ErrAlreadyInitialised
	}

	globalData.log = logger.New("announcer")
	if nil == globalData.log {
		return fault.ErrInvalidLoggerChannel
	}
	globalData.log.Info("starting…")

	globalData.peerTree = avl.New()
	globalData.thisNode = nil
	globalData.change = false

	globalData.peerSet = false
	globalData.rpcsSet = false

	texts, err := net.LookupTXT("node.test.bitmark.com")
	if nil != err {
		return err
	}

	// process DNS entries
	for i, t := range texts {
		t = strings.TrimSpace(t)
		tag, err := parseTag(t)
		if nil != err {
			globalData.log.Infof("ignore TXT[%d]: %q  error: %v", i, t, err)
		} else {
			globalData.log.Infof("process TXT[%d]: %q", i, t)
			globalData.log.Infof("result[%d]: IPv4: %q  IPv6: %q  rpc: %d  connect: %d  subscribe: %d", i, tag.ipv4, tag.ipv6, tag.rpcPort, tag.connectPort, tag.subscribePort)
			globalData.log.Infof("result[%d]: peer public key: %x", i, tag.publicKey)
			globalData.log.Infof("result[%d]: rpc fingerprint: %x", i, tag.certificateFingerprint)

			s1 := util.ConnectionFromIPandPort(tag.ipv4, tag.subscribePort)
			s2 := util.ConnectionFromIPandPort(tag.ipv6, tag.subscribePort)
			c1 := util.ConnectionFromIPandPort(tag.ipv4, tag.connectPort)
			c2 := util.ConnectionFromIPandPort(tag.ipv6, tag.connectPort)

			broadcasts := append(s1.Pack(), s2.Pack()...)
			listeners := append(c1.Pack(), c2.Pack()...)
			globalData.log.Infof("result[%d]: broadcasts: %x  listeners: %x", i, broadcasts, listeners)

			// internal add, as lock is already held
			addPeer(tag.publicKey, broadcasts, listeners)
		}
	}

	if err := globalData.ann.initialise(); nil != err {
		return err
	}

	// all data initialised
	globalData.initialised = true

	// start background processes
	globalData.log.Info("start background…")

	var processes = background.Processes{
		&globalData.ann,
	}

	globalData.background = background.Start(processes, globalData.log)

	return nil
}

// finialise - stop all background tasks
func Finalise() error {
	globalData.Lock()
	defer globalData.Unlock()

	if !globalData.initialised {
		return fault.ErrNotInitialised
	}

	globalData.log.Info("shutting down…")
	globalData.log.Flush()

	// stop background
	globalData.background.Stop()

	// finally...
	globalData.initialised = false

	return nil
}
