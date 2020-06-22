// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/asset"
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/blockheader"
	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/chain"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/ownership"
	"github.com/bitmark-inc/bitmarkd/payment"
	"github.com/bitmark-inc/bitmarkd/peer"
	"github.com/bitmark-inc/bitmarkd/proof"
	"github.com/bitmark-inc/bitmarkd/publish"
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
	theConfiguration, err := getConfiguration(configurationFile)
	if nil != err {
		exitwithstatus.Message("%s: failed to read configuration from: %q  error: %s", program, configurationFile, err)
	}

	// these commands require the configuration and
	// perform enquiries on the configuration
	if len(arguments) > 0 && processConfigCommand(arguments, theConfiguration) {
		return
	}

	// start logging
	if err = logger.Initialise(theConfiguration.Logging); nil != err {
		exitwithstatus.Message("%s: logger setup failed with error: %s", program, err)
	}
	defer logger.Finalise()

	// create a logger channel for the main program
	log := logger.New("main")
	defer log.Info("finished")
	log.Info("starting…")
	log.Infof("version: %s", version)
	log.Debugf("theConfiguration: %v", theConfiguration)

	// ------------------
	// start of real main
	// ------------------

	// optional PID file
	// use if not running under a supervisor program like daemon(8)
	if "" != theConfiguration.PidFile {
		lockFile, err := os.OpenFile(theConfiguration.PidFile, os.O_WRONLY|os.O_EXCL|os.O_CREATE, os.ModeExclusive|0600)
		if err != nil {
			if os.IsExist(err) {
				exitwithstatus.Message("%s: another instance is already running", program)
			}
			exitwithstatus.Message("%s: PID file: %q creation failed, error: %s", program, theConfiguration.PidFile, err)
		}
		fmt.Fprintf(lockFile, "%d\n", os.Getpid())
		lockFile.Close()
		defer os.Remove(theConfiguration.PidFile)
	}

	// set the initial system mode - before any background tasks are started
	err = mode.Initialise(theConfiguration.Chain)
	if nil != err {
		log.Criticalf("mode initialise error: %s", err)
		exitwithstatus.Message("mode initialise error: %s", err)
	}
	defer mode.Finalise()

	// start a profiling http server
	// this uses the default builtin HTTP handler
	// and is not associated with the normal ClientRPC HTTPS server
	if "" != theConfiguration.ProfileHTTP {
		go func() {
			log.Warnf("profile listener on: %s", theConfiguration.ProfileHTTP)
			err = http.ListenAndServe(theConfiguration.ProfileHTTP, nil)
			exitwithstatus.Message("profile error: %s", err)
		}()
	}

	// general info
	log.Infof("test mode: %v", mode.IsTesting())
	log.Infof("database: %q", theConfiguration.Database)

	// connection info
	log.Debugf("%s = %#v", "ClientRPC", theConfiguration.ClientRPC)
	log.Debugf("%s = %#v", "Peering", theConfiguration.Peering)
	log.Debugf("%s = %#v", "Publishing", theConfiguration.Publishing)
	log.Debugf("%s = %#v", "Proofing", theConfiguration.Proofing)

	// start the data storage
	log.Info("initialise storage")
	err = storage.Initialise(theConfiguration.Database.Name, storage.ReadWrite)
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

	// reservoir and block are both ready
	// so can restore any previously saved transactions
	// before any peer services are started
	handles := reservoir.Handles{
		Assets:            storage.Pool.Assets,
		BlockOwnerPayment: storage.Pool.BlockOwnerPayment,
		Transactions:      storage.Pool.Transactions,
		OwnerTxIndex:      storage.Pool.OwnerTxIndex,
		OwnerData:         storage.Pool.OwnerData,
		Shares:            storage.Pool.Shares,
		ShareQuantity:     storage.Pool.ShareQuantity,
	}

	// start the reservoir (verified transaction data cache)
	log.Info("initialise reservoir")
	err = reservoir.Initialise(theConfiguration.CacheDirectory, handles)
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
	if len(arguments) > 0 && processDataCommand(log, arguments, theConfiguration) {
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

	ownership.Initialise(storage.Pool.OwnerList, storage.Pool.OwnerData)

	err = reservoir.LoadFromFile(handles)
	if nil != err && !os.IsNotExist(err) {
		log.Criticalf("reservoir reload error: %s", err)
		exitwithstatus.Message("reservoir reload error: %s", err)
	}

	// network announcements need to be before peer and rpc initialisation
	log.Info("initialise announce")
	nodesDomain := "" // initially none
	switch theConfiguration.Nodes {
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
		nodesDomain = theConfiguration.Nodes // just assume it is a domain name
	}
	err = announce.Initialise(nodesDomain, theConfiguration.CacheDirectory, net.LookupTXT)
	if nil != err {
		log.Criticalf("announce initialise error: %s", err)
		exitwithstatus.Message("announce initialise error: %s", err)
	}
	defer announce.Finalise()

	// start payment services
	err = payment.Initialise(&theConfiguration.Payment)
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

	// start up the peering background processes
	err = peer.Initialise(&theConfiguration.Peering, version, theConfiguration.Fastsync)
	if nil != err {
		log.Criticalf("peer initialise error: %s", err)
		exitwithstatus.Message("peer initialise error: %s", err)
	}
	defer peer.Finalise()

	// start up the publishing background processes
	err = publish.Initialise(&theConfiguration.Publishing, version)
	if nil != err {
		log.Criticalf("publish initialise error: %s", err)
		exitwithstatus.Message("publish initialise error: %s", err)
	}
	defer publish.Finalise()

	// start up the rpc background processes
	err = rpc.Initialise(&theConfiguration.ClientRPC, &theConfiguration.HttpsRPC, version, announce.Get())
	if nil != err {
		log.Criticalf("rpc initialise error: %s", err)
		exitwithstatus.Message("peer initialise error: %s", err)
	}
	defer rpc.Finalise()

	// start proof background processes
	err = proof.Initialise(&theConfiguration.Proofing)
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
