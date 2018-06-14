// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package announce

import (
	"encoding/hex"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/bitmark-inc/bitmarkd/avl"
	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/logger"
)

// type of listener
const (
	TypeRPC  = iota
	TypePeer = iota
)

// type for SHA3 fingerprints
type fingerprintType [32]byte

// RPC entries
type rpcEntry struct {
	address     util.PackedConnection // packed addresses
	fingerprint fingerprintType       // SHA3-256(certificate)
	timestamp   time.Time             // creation time
	local       bool                  // true => never expires
}

// globals for background proccess
type announcerData struct {
	sync.RWMutex // to allow locking

	// logger
	log *logger.L

	// this node's packed annoucements
	publicKey   []byte
	listeners   []byte
	fingerprint fingerprintType
	rpcs        []byte
	peerSet     bool
	rpcsSet     bool

	// tree of nodes available
	peerTree    *avl.Tree
	thisNode    *avl.Node // this node's position in the tree
	treeChanged bool      // tree was changed
	peerFile    string

	// database of all RPCs
	rpcIndex map[fingerprintType]int // index to find rpc entry
	rpcList  []*rpcEntry             // array of RPCs

	// data for thread
	ann announcer

	// for background
	background *background.T

	// set once during initialise
	initialised bool
}

// global data
var globalData announcerData

// initialise the announcement system
// pass a fully qualified domain for root node list
// or empty string for no root nodes
func Initialise(nodesDomain, peerFile string) error {

	globalData.Lock()
	defer globalData.Unlock()

	// no need to start if already started
	if globalData.initialised {
		return fault.ErrAlreadyInitialised
	}

	globalData.log = logger.New("announce")
	globalData.log.Info("starting…")

	globalData.peerTree = avl.New()
	globalData.thisNode = nil
	globalData.treeChanged = false

	globalData.rpcIndex = make(map[fingerprintType]int, 1000)
	globalData.rpcList = make([]*rpcEntry, 0, 1000)

	globalData.peerSet = false
	globalData.rpcsSet = false
	globalData.peerFile = peerFile

	globalData.log.Info("start restoring peer data…")
	if err := restorePeers(globalData.peerFile); err != nil {
		globalData.log.Errorf("fail to restore peer data: %s", err.Error())
	}

	if "" != nodesDomain {
		texts, err := net.LookupTXT(nodesDomain)
		if nil != err {
			return err
		}

		// process DNS entries
		for i, t := range texts {
			t = strings.TrimSpace(t)
			tag, err := parseTag(t)
			if nil != err {
				globalData.log.Infof("ignore TXT[%d]: %q  error: %s", i, t, err)
			} else {
				globalData.log.Infof("process TXT[%d]: %q", i, t)
				globalData.log.Infof("result[%d]: IPv4: %q  IPv6: %q  rpc: %d  connect: %d", i, tag.ipv4, tag.ipv6, tag.rpcPort, tag.connectPort)
				globalData.log.Infof("result[%d]: peer public key: %x", i, tag.publicKey)
				globalData.log.Infof("result[%d]: rpc fingerprint: %x", i, tag.certificateFingerprint)

				listeners := []byte{}

				if nil != tag.ipv4 {
					c1 := util.ConnectionFromIPandPort(tag.ipv4, tag.connectPort)
					listeners = append(listeners, c1.Pack()...)
				}
				if nil != tag.ipv6 {
					c2 := util.ConnectionFromIPandPort(tag.ipv6, tag.connectPort)
					listeners = append(listeners, c2.Pack()...)
				}

				if nil == tag.ipv4 && nil == tag.ipv6 {
					globalData.log.Debugf("result[%d]: ignoring invalid record", i)
				} else {
					globalData.log.Infof("result[%d]: adding: %x", i, listeners)

					// internal add, as lock is already held
					addPeer(tag.publicKey, listeners, 0)
				}
			}
		}
	}
	if err := globalData.ann.initialise(); nil != err {
		return err
	}

	// all data initialised
	globalData.initialised = true

	// start background processes
	globalData.log.Info("start background…")

	processes := background.Processes{
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

	globalData.log.Info("start backing up peer data…")
	if err := backupPeers(globalData.peerFile); err != nil {
		globalData.log.Errorf("fail to backup peer data: %s", err.Error())
	}

	// finally...
	globalData.initialised = false

	globalData.log.Info("finished")
	globalData.log.Flush()

	return nil
}

// convert fingerprint to little endian hex text
func (fingerprint fingerprintType) MarshalText() ([]byte, error) {
	size := hex.EncodedLen(len(fingerprint))
	buffer := make([]byte, size)
	hex.Encode(buffer, fingerprint[:])
	return buffer, nil
}
