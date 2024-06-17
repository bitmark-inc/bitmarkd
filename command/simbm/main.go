// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"os/signal"

	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/exitwithstatus"
	"github.com/bitmark-inc/getoptions"
	"github.com/bitmark-inc/logger"

	//nolint:gocritic // uncomment this to enable profiling
	//"runtime/pprof"
	"syscall"
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
		exitwithstatus.Message("%s: version: %s", program, version)
	}

	if len(options["help"]) > 0 {
		exitwithstatus.Message("usage: %s [--help] [--verbose] [--quiet] --config-file=FILE [[command|help] arguments...]", program)
	}

	if len(options["config-file"]) != 1 {
		exitwithstatus.Message("%s: only one config-file option is required, %d were detected", program, len(options["config-file"]))
	}

	// read options and parse the configuration file
	configurationFile := options["config-file"][0]
	theConfiguration, err := getConfiguration(configurationFile)
	if err != nil {
		exitwithstatus.Message("%s: failed to read configuration from: %q  error: %s", program, configurationFile, err)
	}

	// start logging
	if err = logger.Initialise(theConfiguration.Logging); err != nil {
		exitwithstatus.Message("%s: logger setup failed with error: %s", program, err)
	}
	defer logger.Finalise()

	// create a logger channel for the main program
	log := logger.New("main")
	defer log.Info("shutting down…")
	log.Info("starting…")
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

	// command processing - need lock so do not affect an already running process
	// these commands process data needed for initial setup
	if len(arguments) > 0 {
		processSetupCommand(log, arguments, theConfiguration)
		return
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
	if theConfiguration.Proofer.PublicKey == "" || theConfiguration.Proofer.PrivateKey == "" {
		exitwithstatus.Message("%s: both proofer Public and Private keys must be specified", program)
	}
	publicKey, err := hex.DecodeString(theConfiguration.Proofer.PublicKey)
	if err != nil {
		log.Criticalf("read error on: %s  error: %s", theConfiguration.Proofer.PublicKey, err)
		exitwithstatus.Message("%s: failed reading Public Key: %q  error: %s", program, theConfiguration.Proofer.PublicKey, err)
	}
	privateKey, err := hex.DecodeString(theConfiguration.Proofer.PrivateKey)
	if err != nil {
		log.Criticalf("read error on: %s  error: %s", theConfiguration.Proofer.PrivateKey, err)
		exitwithstatus.Message("%s: failed reading Private Key: %q  error: %s", program, theConfiguration.Proofer.PrivateKey, err)
	}

	// general info
	log.Infof("test mode: %v", mode.IsTesting())

	// keys
	log.Debugf("public key:  %q", publicKey)
	log.Debugf("private key: %q", privateKey)

	// connection info
	log.Debugf("%s = %#v", "Proofer", theConfiguration.Proofer)

	// initialise encryption
	err = zmqutil.StartAuthentication()
	if err != nil {
		log.Criticalf("zmq.AuthStart(): error: %s", err)
		exitwithstatus.Message("%s: zmq.AuthStart() error: %s", program, err)
	}
	log.Warnf("zmq: encryption initialised")

	// start up simulated block publisher
	for i, address := range theConfiguration.Proofer.Publish {
		c, err := util.NewConnection(address)
		if err != nil {
			log.Criticalf("invalid publish[%d]: %q error: %s", i, address, err)
			exitwithstatus.Message("%s: invalid publish[%d]: %q error: %s", program, i, address, err)

		}
		publish, _ := c.CanonicalIPandPort("tcp://")

		log := logger.New("publish")
		Publish(publish, publicKey, privateKey, log)
	}

	// start up simulated block submission
	for i, address := range theConfiguration.Proofer.Submit {
		c, err := util.NewConnection(address)
		if err != nil {
			log.Criticalf("invalid submit[%d]: %q error: %s", i, address, err)
			exitwithstatus.Message("%s: invalid submit[%d]: %q error: %s", program, i, address, err)

		}
		submit, _ := c.CanonicalIPandPort("tcp://")

		slog := logger.New("submit")
		Submission(submit, publicKey, privateKey, slog)
	}

	// erase private key
	privateKey = []byte{}

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
