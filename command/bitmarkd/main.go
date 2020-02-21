// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	net2 "net"
	"os"
	"os/signal"

	"github.com/bitmark-inc/bitmarkd/announce/broadcast"

	//"runtime/pprof"
	"syscall"

	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/asset"
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/blockheader"
	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/chain"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/p2p"
	"github.com/bitmark-inc/bitmarkd/payment"
	"github.com/bitmark-inc/bitmarkd/publish"

	"github.com/bitmark-inc/bitmarkd/consensus"
	"github.com/bitmark-inc/bitmarkd/proof"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/bitmarkd/rpc"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/exitwithstatus"
	"github.com/bitmark-inc/getoptions"
	"github.com/bitmark-inc/logger"
)

// set by the linker: go build -ldflags "-X main.version=M.N" ./...
var version = "zero" // do not change this value

// main program
func main() {
	// ensure exit handler is first
	defer exitwithstatus.Handler()

	flags := []getoptions.Option{
		{Long: "help", HasArg: getoptions.NO_ARGUMENT, Short: 'h'},
		{Long: "verbose", HasArg: getoptions.NO_ARGUMENT, Short: 'v'},
		{Long: "quiet", HasArg: getoptions.NO_ARGUMENT, Short: 'q'},
		{Long: "version", HasArg: getoptions.NO_ARGUMENT, Short: 'V'},
		{Long: "config-file", HasArg: getoptions.REQUIRED_ARGUMENT, Short: 'c'},
		{Long: "memory-stats", HasArg: getoptions.NO_ARGUMENT, Short: 'm'},
	}

	program, options, arguments, err := getoptions.GetOS(flags)
	if nil != err {
		exitwithstatus.Message("%s: getoptions error: %s", program, err)
	}

	if len(options["version"]) > 0 {
		processSetupCommand(program, []string{"version"})
		return
	}

	if len(options["help"]) > 0 {
		processSetupCommand(program, []string{"help"})
		return
	}

	// these commands do not require the configuration and
	// process data needed for initial setup
	if len(arguments) > 0 && processSetupCommand(program, arguments) {
		return
	}

	if 1 != len(options["config-file"]) {
		exitwithstatus.Message("%s: only one config-file option is required, %d were detected", program, len(options["config-file"]))
	}

	// read options and parse the configuration file
	configurationFile := options["config-file"][0]
	masterConfiguration, err := getConfiguration(configurationFile)
	if nil != err {
		exitwithstatus.Message("%s: failed to read configuration from: %q  error: %s", program, configurationFile, err)
	}

	// these commands require the configuration and
	// perform enquiries on the configuration
	if len(arguments) > 0 && processConfigCommand(arguments, masterConfiguration) {
		return
	}

	// start logging
	if err = logger.Initialise(masterConfiguration.Logging); nil != err {
		exitwithstatus.Message("%s: logger setup failed with error: %s", program, err)
	}
	defer logger.Finalise()

	// create a logger channel for the main program
	log := logger.New("main")
	defer log.Info("finished")
	log.Info("starting…")
	log.Infof("version: %s", version)
	log.Debugf("masterConfiguration: %v", masterConfiguration)

	// ------------------
	// start of real main
	// ------------------

	// optional PID file
	// use if not running under a supervisor program like daemon(8)
	if "" != masterConfiguration.PidFile {
		lockFile, err := os.OpenFile(masterConfiguration.PidFile, os.O_WRONLY|os.O_EXCL|os.O_CREATE, os.ModeExclusive|0600)
		if err != nil {
			if os.IsExist(err) {
				exitwithstatus.Message("%s: another instance is already running", program)
			}
			exitwithstatus.Message("%s: PID file: %q creation failed, error: %s", program, masterConfiguration.PidFile, err)
		}
		fmt.Fprintf(lockFile, "%d\n", os.Getpid())
		lockFile.Close()
		defer os.Remove(masterConfiguration.PidFile)
	}

	// set the initial system mode - before any background tasks are started
	err = mode.Initialise(masterConfiguration.Chain)
	if nil != err {
		log.Criticalf("mode initialise error: %s", err)
		exitwithstatus.Message("mode initialise error: %s", err)
	}
	defer mode.Finalise()

	// if requested start profiling
	// if "" != masterConfiguration.ProfileFile {
	// 	f, err := os.Create(masterConfiguration.ProfileFile)
	// 	if nil != err {
	// 		log.Criticalf("cannot open profile output file: '%s'  error: %s", masterConfiguration.ProfileFile, err)
	// 		exitwithstatus.Exit(1)
	// 	}
	// 	defer f.Close()
	// 	pprof.StartCPUProfile(f)
	// 	defer pprof.StopCPUProfile()
	// }

	// general info
	log.Infof("test mode: %v", mode.IsTesting())
	log.Infof("database: %q", masterConfiguration.Database)

	// connection info
	log.Debugf("%s = %#v", "ClientRPC", masterConfiguration.ClientRPC)
	log.Debugf("%s = %#v", "Peering", masterConfiguration.Peering)
	log.Debugf("%s = %#v", "Publishing", masterConfiguration.Publishing)
	log.Debugf("%s = %#v", "Proofing", masterConfiguration.Proofing)

	// start the data storage
	log.Info("initialise storage")
	err = storage.Initialise(masterConfiguration.Database.Name, storage.ReadWrite)
	if nil != err {
		log.Criticalf("storage initialise error: %s", err)
		exitwithstatus.Message("storage initialise error: %s", err)
	}
	defer storage.Finalise()

	if storage.IsMigrationNeed() {
		log.Warn("block database migration required")
	}

	// start asset cache
	err = asset.Initialise()
	if nil != err {
		log.Criticalf("asset initialise error: %s", err)
		exitwithstatus.Message("asset initialise error: %s", err)
	}
	defer asset.Finalise()

	// start the reservoir (verified transaction data cache)
	log.Info("initialise reservoir")

	// reservoir and block are both ready
	// so can restore any previously saved transactions
	// before any peer services are started
	handles := reservoir.Handles{
		Assets:            storage.Pool.Assets,
		BlockOwnerPayment: storage.Pool.BlockOwnerPayment,
		Transaction:       storage.Pool.Transactions,
		OwnerTx:           storage.Pool.OwnerTxIndex,
		OwnerData:         storage.Pool.OwnerData,
		Share:             storage.Pool.ShareQuantity,
		ShareQuantity:     storage.Pool.Shares,
	}
	err = reservoir.Initialise(masterConfiguration.CacheDirectory, handles)
	if nil != err {
		log.Criticalf("reservoir initialise error: %s", err)
		exitwithstatus.Message("reservoir initialise error: %s", err)
	}
	defer reservoir.Finalise()

	// block header data
	log.Info("initialise blockheader")
	err = blockheader.Initialise()
	if nil != err {
		log.Criticalf("blockheader initialise error: %s", err)
		exitwithstatus.Message("blockheader initialise error: %s", err)
	}
	defer blockheader.Finalise()

	log.Info("initialise blockrecord")
	blockrecord.Initialise(storage.Pool.BlockHeaderHash)
	defer blockrecord.Finalise()

	// block data storage - depends on storage and mode
	log.Info("initialise block")
	err = block.Initialise(storage.Pool.Blocks)
	if nil != err {
		log.Criticalf("block initialise error: %s", err)
		exitwithstatus.Message("block initialise error: %s", err)
	}
	defer block.Finalise()

	// these commands are allowed to access the internal database
	if len(arguments) > 0 && processDataCommand(log, arguments, masterConfiguration) {
		return
	}

	// adjust difficulty to fit current status
	height, _, blockHeaderVersion, _ := blockheader.Get()
	if blockrecord.IsDifficultyAppliedVersion(blockHeaderVersion) && difficulty.AdjustTimespanInBlocks < height {
		log.Info("initialise difficulty based on existing blocks")
		_, _, err = blockrecord.AdjustDifficultyAtBlock(blockheader.Height())
		if nil != err {
			log.Criticalf("initialise difficulty error: %s", err)
			exitwithstatus.Message("initialise difficulty error: %s", err)
		}
	}

	err = reservoir.LoadFromFile(handles)
	if nil != err && !os.IsNotExist(err) {
		log.Criticalf("reservoir reload error: %s", err)
		exitwithstatus.Message("reservoir reload error: %s", err)
	}
	// network announcements need to be before peer and rpc initialisation
	log.Info("initialise announce")
	nodesDomain := "" // initially none
	switch masterConfiguration.Nodes {
	case "":
		log.Critical("nodes cannot be blank choose from: none, chain or sub.domain.tld")
		exitwithstatus.Message("nodes cannot be blank choose from: none, chain or sub.domain.tld")
	case "none":
		nodesDomain = "" // nodes disabled
	case "chain":
		switch cn := mode.ChainName(); cn { // ***** FIX THIS: is there a better way?
		case chain.Local:
			nodesDomain = "nodes.localdomain"
		case chain.Testing:
			nodesDomain = "nodes.test.bitmark.com"
		case chain.Bitmark:
			nodesDomain = "nodes.live.bitmark.com"
		default:
			log.Criticalf("unexpected chain name: %q", cn)
			exitwithstatus.Message("unexpected chain name: %q", cn)
		}
	default:
		// domain names are complex to validate so just rely on
		// trying to fetch the TXT records for validation
		nodesDomain = masterConfiguration.Nodes // just assume it is a domain name
	}
	if masterConfiguration.DNSPeerOnly {
		err = announce.Initialise(nodesDomain, masterConfiguration.CacheDirectory, broadcast.DnsOnly, net2.LookupTXT)
	} else {
		err = announce.Initialise(nodesDomain, masterConfiguration.CacheDirectory, broadcast.UsePeers, net2.LookupTXT)
	}

	if nil != err {
		log.Criticalf("announce initialise error: %s", err)
		exitwithstatus.Message("announce initialise error: %s", err)
	}
	defer announce.Finalise()

	// start payment services
	err = payment.Initialise(&masterConfiguration.Payment)
	if nil != err {
		log.Criticalf("payment initialise  error: %s", err)
		exitwithstatus.Message("payment initialise error: %s", err)
	}
	defer payment.Finalise()

	// initialise encryption
	err = zmqutil.StartAuthentication()
	if nil != err {
		log.Criticalf("zmq.AuthStart: error: %s", err)
		exitwithstatus.Message("zmq.AuthStart: error: %s", err)
	}

	if masterConfiguration.DNSPeerOnly {
		err = p2p.Initialise(&masterConfiguration.Peering, version, p2p.DnsOnly)
	} else {
		err = p2p.Initialise(&masterConfiguration.Peering, version, p2p.UsePeers)
	}

	if nil != err {
		log.Criticalf("p2p initialise error: %s", err)
		exitwithstatus.Message("p2p initialise error: %s", err)
	}
	defer p2p.Finalise()

	err = consensus.Initialise(p2p.P2PNode(), masterConfiguration.Fastsync)
	if nil != err {
		log.Criticalf("consensus initialise error: %s", err)
		exitwithstatus.Message("consensus initialise error: %s", err)
	}
	defer consensus.Finalise()

	err = rpc.Initialise(&masterConfiguration.ClientRPC, &masterConfiguration.HttpsRPC, version)
	if nil != err {
		log.Criticalf("rpc initialise error: %s", err)
		exitwithstatus.Message("peer initialise error: %s", err)
	}
	defer rpc.Finalise()

	// start up the publishing background processes

	err = publish.Initialise(&masterConfiguration.Publishing, version)
	if nil != err {
		log.Criticalf("publish initialise error: %s", err)
		exitwithstatus.Message("publish initialise error: %s", err)
	}
	defer publish.Finalise()

	// start proof background processes
	err = proof.Initialise(&masterConfiguration.Proofing)
	if nil != err {
		log.Criticalf("proof initialise error: %s", err)
		exitwithstatus.Message("proof initialise error: %s", err)
	}
	defer proof.Finalise()

	// if memory logging enabled
	if len(options["memory-stats"]) > 0 {
		go memstats()
	}

	// wait for CTRL-C before shutting down to allow manual testing
	if 0 == len(options["quiet"]) {
		fmt.Printf("\n\nWaiting for CTRL-C (SIGINT) or 'kill <pid>' (SIGTERM)…")
	}

	// turn Signals into channel messages
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	sig := <-ch
	log.Infof("received signal: %v", sig)
	if 0 == len(options["quiet"]) {
		fmt.Printf("\nreceived signal: %v\n", sig)
		fmt.Printf("\nshutting down…\n")
	}

	log.Info("shutting down…")
	mode.Set(mode.Stopped)
}
