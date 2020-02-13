// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package announce

import (
	"fmt"
	"path"
	"sync"

	"github.com/bitmark-inc/bitmarkd/announce/rpc"

	"github.com/bitmark-inc/bitmarkd/announce/receptor"

	"github.com/bitmark-inc/bitmarkd/messagebus"

	"github.com/bitmark-inc/bitmarkd/announce/fingerprint"

	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/logger"
	peerlib "github.com/libp2p/go-libp2p-core/peer"
)

type dnsOnlyType bool

const (
	DnsOnly  dnsOnlyType = true
	UsePeers dnsOnlyType = false
)

// type of listener
const (
	TypeRPC  = iota
	TypePeer = iota
)

// file for storing saves peers
const backupFile = "peers.json"

// globals for background process
type announcerData struct {
	sync.RWMutex // to allow locking

	// logger
	log *logger.L

	// this node's packed annoucements
	fingerprint fingerprint.Type

	// tree of nodes available
	receptors receptor.Receptor

	backupFile string

	// database of all RPCs
	rpcs rpc.RPC

	// data for thread
	ann announcer

	nodesLookup lookup

	// for background
	background *background.T

	// set once during initialise
	initialised bool
	// only use dns record as peer nodes
	dnsPeerOnly dnsOnlyType
}

// global data
var globalData announcerData

// format for timestamps
const timeFormat = "2006-01-02 15:04:05"

// Initialise - set up the announcement system
// pass a fully qualified domain for root node list
// or empty string for no root nodes
func Initialise(nodesDomain, cacheDirectory string, dnsPeerOnly dnsOnlyType, f func(string) ([]string, error)) error {

	globalData.Lock()
	defer globalData.Unlock()

	// no need to start if already started
	if globalData.initialised {
		return fault.AlreadyInitialised
	}

	globalData.log = logger.New("announce")
	globalData.log.Info("starting…")

	globalData.receptors = receptor.New(globalData.log)
	globalData.rpcs = rpc.New()

	globalData.backupFile = path.Join(cacheDirectory, backupFile)

	globalData.dnsPeerOnly = dnsPeerOnly

	globalData.log.Info("start restoring peer data…")
	if globalData.dnsPeerOnly == UsePeers { //disable restore to avoid restore non-dns node
		if list, err := receptor.Restore(globalData.backupFile); err == nil {
			for _, item := range list.Receptors {
				id, err := peerlib.IDFromBytes(item.ID)
				addrs := util.GetMultiAddrsFromBytes(item.Listeners.Address)
				if err != nil || nil != addrs {
					continue
				}
				util.LogDebug(globalData.log, util.CoReset, fmt.Sprintf("restore peer ID:%s", id.ShortString()))

				globalData.receptors.Add(id, addrs, item.Timestamp)
				globalData.receptors.Tree().Print(false)
			}
		} else {
			globalData.log.Errorf("fail to restore peer data: %s", err.Error())
		}
	}

	if err := globalData.nodesLookup.initialise(nodesDomain, f); nil != err {
		return err
	}

	if err := globalData.ann.initialise(); nil != err {
		return err
	}

	// all data initialised
	globalData.initialised = true

	// start background processes
	globalData.log.Info("start background…")

	processes := background.Processes{
		&globalData.nodesLookup, &globalData.ann,
	}

	globalData.background = background.Start(processes, messagebus.Bus.Announce.Chan())

	return nil
}

// Finalise - stop all background tasks
func Finalise() error {

	if !globalData.initialised {
		return fault.NotInitialised
	}

	globalData.log.Info("shutting down…")
	globalData.log.Flush()

	// stop background
	globalData.background.Stop()

	// release message bus
	messagebus.Bus.Announce.Release()

	globalData.log.Info("start backing up peer data…")
	if err := receptor.Backup(globalData.backupFile, globalData.receptors.Tree()); err != nil {
		globalData.log.Errorf("fail to backup peer data: %s", err.Error())
	}

	// finally...
	globalData.initialised = false

	globalData.log.Info("finished")
	globalData.log.Flush()

	return nil
}
