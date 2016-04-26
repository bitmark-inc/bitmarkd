// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"crypto/tls"
	"fmt"
	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/configuration"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/mine"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/payment"
	"github.com/bitmark-inc/bitmarkd/peer"
	"github.com/bitmark-inc/bitmarkd/pool"
	"github.com/bitmark-inc/bitmarkd/rpc"
	"github.com/bitmark-inc/bitmarkd/transaction"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/bitmarkd/version"
	"github.com/bitmark-inc/exitwithstatus"
	"github.com/bitmark-inc/getoptions"
	"github.com/bitmark-inc/listener"
	"github.com/bitmark-inc/logger"
	"os"
	"os/signal"
	//"runtime/pprof"
	"syscall"
	"time"
)

type serverChannel struct {
	// initial values
	limit               int
	addresses           []string
	certificateFileName string
	keyFileName         string
	callback            listener.Callback
	argument            interface{}

	// filled in later
	tlsConfiguration *tls.Config
	limiter          *listener.Limiter
	listener         *listener.MultiListener
}

func main() {
	// ensure exit handler is first
	defer exitwithstatus.Handler()

	flags := []getoptions.Option{
		{"help", getoptions.NO_ARGUMENT, 'h'},
		{"verbose", getoptions.NO_ARGUMENT, 'v'},
		{"quiet", getoptions.NO_ARGUMENT, 'q'},
		{"version", getoptions.NO_ARGUMENT, 'V'},
		{"config-file", getoptions.REQUIRED_ARGUMENT, 'c'},
	}

	program, options, arguments, err := getoptions.GetOS(flags)
	if nil != err {
		exitwithstatus.Message("%s: getoptions error: %v", program, err)
	}

	if len(options["version"]) > 0 {
		exitwithstatus.Message("%s: version: %s", program, version.Version)
	}

	if len(options["help"]) > 0 {
		exitwithstatus.Message("usage: %s [--help] [--verbose] [--quiet] --config-file=FILE [[command|help] arguments...]", program)
	}

	if 1 != len(options["config-file"]) {
		exitwithstatus.Message("%s: only one config-file option is required, %d were detected", program, len(options["config-file"]))
	}

	// read options and parse the configuration file
	configurationFile := options["config-file"][0]
	masterConfiguration, err := configuration.GetConfiguration(configurationFile)
	if nil != err {
		exitwithstatus.Message("%s: failed to read configuration from: %q  error: %v", program, configurationFile, err)
	}

	// start logging
	if err = logger.Initialise(masterConfiguration.Logging.File, masterConfiguration.Logging.Size, masterConfiguration.Logging.Count); nil != err {
		exitwithstatus.Message("%s: logger setup failed with error: %v", err)
	}
	defer logger.Finalise()
	logger.LoadLevels(masterConfiguration.Logging.Levels)

	// create a logger channel for the main program
	log := logger.New("main")
	defer log.Info("shutting down…")
	log.Info("starting…")
	log.Debugf("masterConfiguration: %v", masterConfiguration)

	// set up the fault panic log (now that logging is available
	fault.Initialise()
	defer fault.Finalise()

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
			exitwithstatus.Message("%s: PID file: %q creation failed, error: %v", program, masterConfiguration.PidFile, err)
		}
		fmt.Fprintf(lockFile, "%d\n", os.Getpid())
		lockFile.Close()
		defer os.Remove(masterConfiguration.PidFile)
	}

	// command processing - need lock so do not affect an already running process
	// these commands process data needed for initial setup
	if len(arguments) > 0 && processSetupCommand(log, arguments, masterConfiguration) {
		return
	}

	// // if requested start profiling
	// if "" != masterConfiguration.ProfileFile {
	// 	f, err := os.Create(masterConfiguration.ProfileFile)
	// 	if nil != err {
	// 		log.Criticalf("cannot open profile output file: '%s'  error: %v", masterConfiguration.ProfileFile, err)
	// 		exitwithstatus.Exit(1)
	// 	}
	// 	defer f.Close()
	// 	pprof.StartCPUProfile(f)
	// 	defer pprof.StopCPUProfile()
	// }

	// set the initial system mode - before any background tasks are started
	mode.Initialise(masterConfiguration.Chain)
	defer mode.Finalise()

	// ensure keys are set
	if "" == masterConfiguration.Peering.PublicKey || "" == masterConfiguration.Peering.PrivateKey {
		exitwithstatus.Message("%s: both peering Public and Private keys must be specified", program)
	}
	publicKey, err := readKeyFile(masterConfiguration.Peering.PublicKey)
	if nil != err {
		log.Criticalf("read error on: %s  error: %v", masterConfiguration.Peering.PublicKey, err)
		exitwithstatus.Message("%s: failed reading Public Key: %q  error: %v", program, masterConfiguration.Peering.PublicKey, err)
	}
	privateKey, err := readKeyFile(masterConfiguration.Peering.PrivateKey)
	if nil != err {
		log.Criticalf("read error on: %s  error: %v", masterConfiguration.Peering.PrivateKey, err)
		exitwithstatus.Message("%s: failed reading Private Key: %q  error: %v", program, masterConfiguration.Peering.PrivateKey, err)
	}

	// general info
	log.Infof("test mode: %v", mode.IsTesting())
	log.Infof("database: %q", masterConfiguration.Database)

	// keys
	log.Debugf("public key:  %q", publicKey)
	log.Debugf("private key: %q", privateKey)

	// connection info
	log.Debugf("%s = %#v", "ClientRPC", masterConfiguration.ClientRPC)
	log.Debugf("%s = %#v", "Peering", masterConfiguration.Peering)
	log.Debugf("%s = %#v", "Mining", masterConfiguration.Mining)

	// start the memory pool
	log.Info("start pool")
	pool.Initialise(masterConfiguration.Database.Name)
	defer pool.Finalise()

	// block data storage - depends on pool
	log.Info("initialise block")
	block.Initialise()
	defer block.Finalise()

	// transaction data storage - depends on pool
	log.Info("initialise transaction")
	transaction.Initialise()
	defer transaction.Finalise()

	// these commands are allowed to access the internal database
	if len(arguments) > 0 && processDataCommand(log, arguments, masterConfiguration) {
		return
	}

	// network announcements - depends on pool
	log.Info("initialise announce")
	announce.Initialise()
	defer announce.Finalise()

	// various logs
	rpcLog := logger.New("rpc-server")
	if nil == rpcLog {
		log.Critical("failed to create rpcLog")
		exitwithstatus.Exit(1)
	}
	mineLog := logger.New("mine-server")
	if nil == mineLog {
		log.Critical("failed to create mineLog")
		exitwithstatus.Exit(1)
	}

	servers := map[string]*serverChannel{
		"rpc": {
			limit:               masterConfiguration.ClientRPC.MaximumConnections,
			addresses:           masterConfiguration.ClientRPC.Listen,
			certificateFileName: masterConfiguration.ClientRPC.Certificate,
			keyFileName:         masterConfiguration.ClientRPC.PrivateKey,
			callback:            rpc.Callback,
			argument: &rpc.ServerArgument{
				Log:       rpcLog,
				StartTime: time.Now().UTC(),
			},
		},
		"mine": {
			limit:               masterConfiguration.Mining.MaximumConnections,
			addresses:           masterConfiguration.Mining.Listen,
			certificateFileName: masterConfiguration.Mining.Certificate,
			keyFileName:         masterConfiguration.Mining.PrivateKey,
			callback:            mine.Callback,
			argument: &mine.ServerArgument{
				Log: mineLog,
			},
		},
	}

	// capture a set of this certificate fingerprints
	myFingerprints := make(map[util.FingerprintBytes]bool)

	// validate server parameters
	for name, server := range servers {
		log.Infof("validate: %s", name)
		fingerprint, ok := verifyListen(log, name, server)
		if !ok {
			log.Criticalf("invalid %s parameters", name)
			exitwithstatus.Exit(1)
		}
		if 0 == server.limit {
			continue
		}
		log.Infof("multi listener for: %s", name)
		ml, err := listener.NewMultiListener(name, server.addresses, server.tlsConfiguration, server.limiter, server.callback)
		if nil != err {
			log.Criticalf("invalid %s listen addresses", name)
			exitwithstatus.Exit(1)
		}
		server.listener = ml
		myFingerprints[*fingerprint] = true

		peerData := announce.PeerData{
			// Type:        announce.TypeNone,
			// State:       announce.StateAllowed,
			Fingerprint: fingerprint,
		}
		switch name {
		case "rpc":
			for _, address := range masterConfiguration.ClientRPC.Announce {
				announce.AddPeer(address, announce.TypeRPC, &peerData)
			}
		case "peer":
			for _, address := range masterConfiguration.Peering.Announce {
				announce.AddPeer(address, announce.TypePeer, &peerData)
			}
		case "mine":
			// no need to announce - currently assume only certain nodes will be mines (i.e. have attached miners)
			// and these will be local connections or be run as pools
		default:
			log.Criticalf("invalid server type: %s", name)
			exitwithstatus.Exit(1)
		}
	}

	// connect to various payment services
	err = payment.BitcoinInitialise(masterConfiguration.Bitcoin)
	if nil != err {
		log.Criticalf("failed to initialise Bitcoin  error: %v", err)
		exitwithstatus.Exit(1)
	}
	defer payment.BitcoinFinalise()

	// start up the peering
	err = peer.Initialise(masterConfiguration.Peering.Listen, mode.ChainName(), publicKey, privateKey)
	if nil != err {
		log.Criticalf("failed to initialise peer  error: %v", err)
		exitwithstatus.Exit(1)
	}
	defer peer.Finalise()
	privateKey = ""

	// now start listeners - these can access memory pools
	serversStarted := 0
	for name, server := range servers {
		if nil != server.listener {
			log.Infof("starting server: %s  with: %v", name, server.argument)
			server.listener.Start(server.argument)
			defer server.listener.Stop()
			serversStarted += 1
		}
	}
	if 0 == serversStarted {
		log.Critical("no servers started")
		exitwithstatus.Exit(1)
	}

	// start up p2p clients
	for i, remote := range masterConfiguration.Peering.Connect {
		//err := p2p.Add(address)
		err := peer.ConnectTo(remote.PublicKey, remote.Address)
		if nil != err {
			log.Warnf("client: %d failed to connect to: %s error: %v", i, remote.Address, err)
			continue
		}
		log.Infof("client: %d connected to %s", i, remote.Address)
	}

	// start mining background processes
	mine.Initialise()
	defer mine.Finalise()

	// wait for CTRL-C before shutting down to allow manual testing
	if 0 == len(options["quiet"]) {
		fmt.Printf("\n\nWaiting for CTRL-C (SIGINT) or 'kill <pid>' (SIGTERM)…")
	}

	// turn Signals into channel messages
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	sig := <-ch
	log.Infof("received signal: %v", sig)
	if 0 == len(options["quiet"]) {
		fmt.Printf("\nreceived signal: %v\n", sig)
		fmt.Printf("\nshutting down...\n")
	}
}
