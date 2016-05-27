// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/bitmarkd/version"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/exitwithstatus"
	"github.com/bitmark-inc/getoptions"
	"github.com/bitmark-inc/logger"
	"os"
	"os/signal"
	//"runtime/pprof"
	"syscall"
	//"time"
)

// bitmark minerd main program
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
	if len(arguments) > 0 {
		processSetupCommand(log, arguments, masterConfiguration)
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
	publicKey, err := zmqutil.ReadKeyFile(masterConfiguration.Peering.PublicKey)
	if nil != err {
		log.Criticalf("read error on: %s  error: %v", masterConfiguration.Peering.PublicKey, err)
		exitwithstatus.Message("%s: failed reading Public Key: %q  error: %v", program, masterConfiguration.Peering.PublicKey, err)
	}
	privateKey, err := zmqutil.ReadKeyFile(masterConfiguration.Peering.PrivateKey)
	if nil != err {
		log.Criticalf("read error on: %s  error: %v", masterConfiguration.Peering.PrivateKey, err)
		exitwithstatus.Message("%s: failed reading Private Key: %q  error: %v", program, masterConfiguration.Peering.PrivateKey, err)
	}

	// general info
	log.Infof("test mode: %v", mode.IsTesting())

	// keys
	log.Debugf("public key:  %q", publicKey)
	log.Debugf("private key: %q", privateKey)

	// connection info
	log.Debugf("%s = %#v", "Peering", masterConfiguration.Peering)

	// internal queues
	ProofProxy()
	SubmitQueue()

	// start background processes
	// these will has blocks, changing nonce to meet difficulty
	// then submit a block to the right bitmarkd for verification
	for i := 1; i <= masterConfiguration.Threads; i += 1 {
		prflog := logger.New(fmt.Sprintf("proofer-%d", i))
		err := ProofThread(prflog)
		if nil != err {
			log.Criticalf("proof[%d]: error: %v", i, err)
			exitwithstatus.Message("%s: proof[%d]: error: %v", program, i, err)
		}
	}

	//proofer.Initialise(masterConfiguration.Threads, prflog)
	//defer proofer.Finalise()

	// initialise encryption
	err = zmqutil.StartAuthentication()
	if nil != err {
		log.Criticalf("zmq.AuthStart(): error: %v", err)
		exitwithstatus.Message("%s: zmq.AuthStart() error: %v", program, err)
	}

	clientCount := 0
	// start up bitmarkd clients these subscribe to bitmarkd
	// blocks publisher to obtain blocks for mining
	for i, remote := range masterConfiguration.Peering.Connect {
		blocksAddress, err := util.CanonicalIPandPort("tcp://", remote.Blocks)
		if nil != err {
			log.Warnf("client: %d invalid blocks publisher: %q error: %v", i, remote.Blocks, err)
			continue
		}

		submitAddress, err := util.CanonicalIPandPort("tcp://", remote.Submit)
		if nil != err {
			log.Warnf("client: %d invalid submit address: %q error: %v", i, remote.Submit, err)
			continue
		}

		log.Infof("client: %d subscribe: %q  submit: %q", i, remote.Blocks, remote.Submit)

		mlog := logger.New(fmt.Sprintf("submitter-%d", i))
		err = Submitter(i, submitAddress, remote.PublicKey, publicKey, privateKey, mlog)
		if nil != err {
			log.Warnf("submitter: %d failed error: %v", i, err)
			continue
		}

		slog := logger.New(fmt.Sprintf("subscriber-%d", i))
		err = Subscribe(i, blocksAddress, remote.PublicKey, publicKey, privateKey, slog)
		if nil != err {
			log.Warnf("subscribe: %d failed error: %v", i, err)
			continue
		}
		clientCount += 1
	}

	// erase the private key from memory
	privateKey = ""

	// abort if no clients were connected

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
