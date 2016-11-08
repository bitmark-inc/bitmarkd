// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"crypto/tls"
	"fmt"
	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/asset"
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/blockring"
	"github.com/bitmark-inc/bitmarkd/chain"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/payment"
	"github.com/bitmark-inc/bitmarkd/peer"
	"github.com/bitmark-inc/bitmarkd/proof"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/bitmarkd/rpc"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/bitmarkd/version"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
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
		{Long: "help", HasArg: getoptions.NO_ARGUMENT, Short: 'h'},
		{Long: "verbose", HasArg: getoptions.NO_ARGUMENT, Short: 'v'},
		{Long: "quiet", HasArg: getoptions.NO_ARGUMENT, Short: 'q'},
		{Long: "version", HasArg: getoptions.NO_ARGUMENT, Short: 'V'},
		{Long: "config-file", HasArg: getoptions.REQUIRED_ARGUMENT, Short: 'c'},
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
	masterConfiguration, err := getConfiguration(configurationFile)
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
	defer log.Info("finished")
	log.Info("starting…")
	log.Debugf("masterConfiguration: %v", masterConfiguration)

	// set up the fault panic log (now that logging is available)
	err = fault.Initialise()
	if nil != err {
		log.Criticalf("fault initialise error: %v", err)
		exitwithstatus.Message("fault initialise error: %v", err)
	}
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
	err = mode.Initialise(masterConfiguration.Chain)
	if nil != err {
		log.Criticalf("mode initialise error: %v", err)
		exitwithstatus.Message("mode initialise error: %v", err)
	}
	defer mode.Finalise()

	// general info
	log.Infof("test mode: %v", mode.IsTesting())
	log.Infof("database: %q", masterConfiguration.Database)

	// connection info
	log.Debugf("%s = %#v", "ClientRPC", masterConfiguration.ClientRPC)
	log.Debugf("%s = %#v", "Peering", masterConfiguration.Peering)
	log.Debugf("%s = %#v", "Proofing", masterConfiguration.Proofing)

	// start the data storage
	log.Info("initialise storage")
	err = storage.Initialise(masterConfiguration.Database.Name)
	if nil != err {
		log.Criticalf("storage initialise error: %v", err)
		exitwithstatus.Message("storage initialise error: %v", err)
	}
	defer storage.Finalise()

	// start the reservoir (verified transaction data cache)
	log.Info("initialise reservoir")
	err = reservoir.Initialise()
	if nil != err {
		log.Criticalf("reservoir initialise error: %v", err)
		exitwithstatus.Message("reservoir initialise error: %v", err)
	}
	defer reservoir.Finalise()

	// start asset cache
	err = asset.Initialise()
	if nil != err {
		log.Criticalf("asset initialise error: %v", err)
		exitwithstatus.Message("asset initialise error: %v", err)
	}
	defer asset.Finalise()

	// block hash ring buffer
	log.Info("initialise blockring")
	err = blockring.Initialise()
	if nil != err {
		log.Criticalf("blockring initialise error: %v", err)
		exitwithstatus.Message("blockring initialise error: %v", err)
	}
	defer blockring.Finalise()

	// block data storage - depends on storage and mode
	log.Info("initialise block")
	err = block.Initialise()
	if nil != err {
		log.Criticalf("block initialise error: %v", err)
		exitwithstatus.Message("block initialise error: %v", err)
	}
	defer block.Finalise()

	// these commands are allowed to access the internal database
	if len(arguments) > 0 && processDataCommand(log, arguments, masterConfiguration) {
		return
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
	err = announce.Initialise(nodesDomain)
	if nil != err {
		log.Criticalf("announce initialise error: %v", err)
		exitwithstatus.Message("announce initialise error: %v", err)
	}
	defer announce.Finalise()

	// various logs
	rpcLog := logger.New("rpc-server")
	if nil == rpcLog {
		log.Critical("failed to create rpcLog")
		exitwithstatus.Message("failed to create rpcLog")
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
	}

	// validate server parameters
	for name, server := range servers {
		log.Infof("validate: %s", name)
		certificate, ok := verifyListen(log, name, server)
		if !ok {
			log.Criticalf("invalid %s parameters", name)
			exitwithstatus.Message("invalid %s parameters", name)
		}
		if 0 == server.limit {
			continue
		}
		log.Infof("multi listener for: %s", name)
		ml, err := listener.NewMultiListener(name, server.addresses, server.tlsConfiguration, server.limiter, server.callback)
		if nil != err {
			log.Criticalf("invalid %s listen addresses", name)
			exitwithstatus.Message("invalid %s listen addresses", name)
		}
		server.listener = ml

		fingerprint := CertificateFingerprint(certificate)
		log.Infof("%s: SHA3-256 fingerprint: %x", name, fingerprint)

		// store certificate
		announce.AddCertificate(fingerprint, certificate)

		switch name {
		case "rpc":
			rpcs := make([]byte, 0, 100) // ***** FIX THIS: need a better default size
			for _, address := range masterConfiguration.ClientRPC.Announce {
				c, err := util.NewConnection(address)
				if nil != err {
					log.Criticalf("invalid %s listen announce: %q  error: %v", name, address, err)
					exitwithstatus.Message("invalid %s listen announce: %q  error: %v", name, address, err)
				}
				rpcs = append(rpcs, c.Pack()...)
			}
			err := announce.SetRPC(fingerprint, rpcs)
			if nil != err {
				log.Criticalf("announce.SetRPC error: %v", err)
				exitwithstatus.Message("announce.SetRPC error: %v", err)
			}
		}
	}

	// start payment services
	paymentConfiguration := &payment.Configuration{
		Bitcoin: &masterConfiguration.Bitcoin,
	}
	err = payment.Initialise(paymentConfiguration)
	if nil != err {
		log.Criticalf("payment initialise  error: %v", err)
		exitwithstatus.Message("payment initialise error: %v", err)
	}
	defer payment.Finalise()

	// initialise encryption
	err = zmqutil.StartAuthentication()
	if nil != err {
		log.Criticalf("zmq.AuthStart: error: %v", err)
		exitwithstatus.Message("zmq.AuthStart: error: %v", err)
	}

	// start up the peering background processes
	err = peer.Initialise(&masterConfiguration.Peering)
	if nil != err {
		log.Criticalf("peer initialise error: %v", err)
		exitwithstatus.Message("peer initialise error: %v", err)
	}
	defer peer.Finalise()

	// now start rpc listeners - these can access memory pools
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
		log.Critical("no RPC servers started")
		exitwithstatus.Message("no RPC servers started")
	}

	// start proof background processes
	err = proof.Initialise(&masterConfiguration.Proofing)
	if nil != err {
		log.Criticalf("proof initialise error: %v", err)
		exitwithstatus.Message("proof initialise error: %v", err)
	}
	defer proof.Finalise()

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
		fmt.Printf("\nshutting down…\n")
	}
	log.Info("shutting down…")
}
