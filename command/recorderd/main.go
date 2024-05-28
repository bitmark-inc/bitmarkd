// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/util"
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
	}

	program, options, arguments, err := getoptions.GetOS(flags)
	if err != nil {
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

	// command processing - need lock so do not affect an already running process
	// these commands don't require the configuration and
	// process data needed for initial setup
	if len(arguments) > 0 {
		if processSetupCommand(program, arguments) {
			return
		}
	}

	if len(options["config-file"]) != 1 {
		exitwithstatus.Message("%s: only one config-file option is required, %d were detected", program, len(options["config-file"]))
	}

	// read options and parse the configuration file
	configurationFile := options["config-file"][0]

	reader := newConfigReader()

	rescheduleChannel := make(chan struct{})
	calendar := newJobCalendar(rescheduleChannel)
	reader.SetCalendar(calendar)

	err = reader.FirstRefresh(configurationFile)
	if err != nil {
		exitwithstatus.Message("%s: failed to read configuration from: %q  error: %s", program, configurationFile, err)
	}

	theConfiguration, err := reader.GetConfig()
	if err != nil {
		exitwithstatus.Message("%s: configuration is not found", program)
	}

	// start logging
	if err = logger.Initialise(theConfiguration.Logging); err != nil {
		exitwithstatus.Message("%s: logger setup failed with error: %s", program, err)
	}
	defer logger.Finalise()

	watcherChannel := WatcherChannel{
		change: make(chan struct{}, 1),
		remove: make(chan struct{}, 1),
	}
	watcher, err := newFileWatcher(configurationFile, logger.New(FileWatcherLoggerPrefix), watcherChannel)
	if err != nil {
		exitwithstatus.Message("%s: file watcher setup failed with error: %s",
			program,
			err,
		)
	}

	reader.SetWatcher(watcher)

	configLogger := logger.New(ReaderLoggerPrefix)
	err = reader.SetLog(configLogger)
	if err != nil {
		exitwithstatus.Message("%s: new logger '%s' failed with error: %s", program, ReaderLoggerPrefix, err)
	}

	calendarLogger := logger.New(jobCalendarPrefix)
	calendar.SetLog(calendarLogger)

	// create a logger channel for the main program
	log := logger.New("main")
	defer log.Info("shutting down…")
	log.Info("starting…")
	log.Infof("version: %s", version)
	log.Debugf("theConfiguration: %v", theConfiguration)

	// ------------------
	// start of real main
	// ------------------

	// optional PID file
	// use if not running under a supervisor program like daemon(8)
	if theConfiguration.PidFile != "" {
		lockFile, err := os.OpenFile(theConfiguration.PidFile, os.O_WRONLY|os.O_EXCL|os.O_CREATE, os.ModeExclusive|0o600)
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

	// // if requested start profiling
	// if "" != theConfiguration.ProfileFile {
	// 	f, err := os.Create(theConfiguration.ProfileFile)
	// 	if err != nil {
	// 		log.Criticalf("cannot open profile output file: '%s'  error: %s", theConfiguration.ProfileFile, err)
	// 		exitwithstatus.Exit(1)
	// 	}
	// 	defer f.Close()
	// 	pprof.StartCPUProfile(f)
	// 	defer pprof.StopCPUProfile()
	// }

	// set the initial system mode - before any background tasks are started
	mode.Initialise(theConfiguration.Chain)
	defer mode.Finalise()

	// ensure keys are set
	if theConfiguration.Peering.PublicKey == "" || theConfiguration.Peering.PrivateKey == "" {
		exitwithstatus.Message("%s: both peering Public and Private keys must be specified", program)
	}
	publicKey, err := zmqutil.ReadPublicKey(theConfiguration.Peering.PublicKey)
	if err != nil {
		log.Criticalf("read error on: %s  error: %s", theConfiguration.Peering.PublicKey, err)
		exitwithstatus.Message("%s: failed reading Public Key: %q  error: %s", program, theConfiguration.Peering.PublicKey, err)
	}
	privateKey, err := zmqutil.ReadPrivateKey(theConfiguration.Peering.PrivateKey)
	if err != nil {
		log.Criticalf("read error on: %s  error: %s", theConfiguration.Peering.PrivateKey, err)
		exitwithstatus.Message("%s: failed reading Private Key: %q  error: %s", program, theConfiguration.Peering.PrivateKey, err)
	}

	// general info
	log.Infof("test mode: %v", mode.IsTesting())

	// keys
	log.Tracef("public key:  %x", publicKey)
	log.Tracef("private key: %x", privateKey)

	// connection info
	log.Debugf("%s = %#v", "Peering", theConfiguration.Peering)

	// internal queues
	ProofProxy()
	SubmitQueue()

	proofer := newProofer(logger.New(prooferLoggerPrefix), reader)
	reader.SetProofer(proofer)
	// start background processes
	// these will has blocks, changing nonce to meet difficulty
	// then submit a block to the right bitmarkd for verification
	proofer.StartHashing()

	// initialise encryption
	err = zmqutil.StartAuthentication()
	if err != nil {
		log.Criticalf("zmq.AuthStart(): error: %s", err)
		exitwithstatus.Message("%s: zmq.AuthStart() error: %s", program, err)
	}
	managerLogger := logger.New(JobManagerPrefix)
	jobManager := newJobManager(
		calendar,
		proofer,
		rescheduleChannel,
		managerLogger,
	)
	jobManager.Start()

	watcher.Start()

	reader.FirstTimeRun()
	reader.Start()

	// start up bitmarkd clients these subscribe to bitmarkd
	// blocks publisher to obtain blocks for mining
connection_setup:
	for i, remote := range theConfiguration.Peering.Connect {

		serverPublicKey, err := zmqutil.ReadPublicKey(remote.PublicKey)
		if err != nil {
			log.Warnf("client: %d invalid server publickey: %q error: %s", i, remote.PublicKey, err)
			continue connection_setup
		}

		bc, err := util.NewConnection(remote.Blocks)
		if err != nil {
			log.Warnf("client: %d invalid blocks publisher: %q error: %s", i, remote.Blocks, err)
			continue connection_setup
		}
		blocksAddress, blocksv6 := bc.CanonicalIPandPort("tcp://")

		sc, err := util.NewConnection(remote.Submit)
		if err != nil {
			log.Warnf("client: %d invalid submit address: %q error: %s", i, remote.Submit, err)
			continue connection_setup
		}
		submitAddress, submitv6 := sc.CanonicalIPandPort("tcp://")

		log.Infof("client: %d subscribe: %q  submit: %q", i, remote.Blocks, remote.Submit)

		mlog := logger.New(fmt.Sprintf("submitter-%d", i))
		err = Submitter(i, submitAddress, submitv6, serverPublicKey, publicKey, privateKey, mlog)
		if err != nil {
			log.Warnf("submitter: %d failed error: %s", i, err)
			continue connection_setup
		}

		slog := logger.New(fmt.Sprintf("subscriber-%d", i))
		err = Subscribe(i, blocksAddress, blocksv6, serverPublicKey, publicKey, privateKey, slog, proofer)
		if err != nil {
			log.Warnf("subscribe: %d failed error: %s", i, err)
			continue connection_setup
		}
	}

	// erase the private key from memory
	//lint:ignore SA4006 we want to make sure we clean privateKey
	privateKey = []byte{}

	// abort if no clients were connected

	// wait for CTRL-C before shutting down to allow manual testing
	if len(options["quiet"]) == 0 {
		fmt.Printf("\n\nWaiting for CTRL-C (SIGINT) or 'kill <pid>' (SIGTERM)…")
	}

	// turn Signals into channel messages
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	sig := <-ch
	log.Infof("received signal: %v", sig)
	if len(options["quiet"]) == 0 {
		fmt.Printf("\nreceived signal: %v\n", sig)
		fmt.Printf("\nshutting down...\n")
	}
}
