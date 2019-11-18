// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bitmark-inc/bitmarkd/p2p"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"

	"github.com/multiformats/go-multiaddr"

	"github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-core/crypto"

	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/exitwithstatus"
	"github.com/bitmark-inc/getoptions"
	"github.com/bitmark-inc/logger"
)

// set by the linker: go build -ldflags "-X main.version=M.N" ./...
var version = "zero" // do not change this value

const (
	recorderdProtocol = "/recorderd/1.0.0"
	maxBytes          = 3000
)

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
	if nil != err {
		exitwithstatus.Message("%s: getoptions error: %s", program, err)
	}

	if len(options["version"]) > 0 {
		exitwithstatus.Message("%s: version: %s", program, version)
	}

	if len(options["help"]) > 0 {
		exitwithstatus.Message("usage: %s [--help] [--verbose] [--quiet] --config-file=FILE [[command|help] arguments...]", program)
	}

	// command processing - need lock so do not affect an already running process
	// these commands don't require the configuration and
	// process data needed for initial setup
	if len(arguments) > 0 {
		processSetupCommand(arguments)
		return
	}

	if 1 != len(options["config-file"]) {
		exitwithstatus.Message("%s: only one config-file option is required, %d were detected", program, len(options["config-file"]))
	}

	// read options and parse the configuration file
	configurationFile := options["config-file"][0]

	reader := newConfigReader()

	rescheduleChannel := make(chan struct{})
	calendar := newJobCalendar(rescheduleChannel)
	reader.SetCalendar(calendar)

	err = reader.FirstRefresh(configurationFile)
	if nil != err {
		exitwithstatus.Message("%s: failed to read configuration from: %q  error: %s", program, configurationFile, err)
	}

	masterConfiguration, err := reader.GetConfig()
	if nil != err {
		exitwithstatus.Message("%s: configuration is not found", program)
	}

	// start logging
	if err = logger.Initialise(masterConfiguration.Logging); nil != err {
		exitwithstatus.Message("%s: logger setup failed with error: %s", program, err)
	}
	defer logger.Finalise()

	watcherChannel := WatcherChannel{
		change: make(chan struct{}, 1),
		remove: make(chan struct{}, 1),
	}
	watcher, err := newFileWatcher(configurationFile, logger.New(FileWatcherLoggerPrefix), watcherChannel)
	if nil != err {
		exitwithstatus.Message("%s: file watcher setup failed with error: %s",
			program,
			err,
		)
	}

	reader.SetWatcher(watcher)

	configLogger := logger.New(ReaderLoggerPrefix)
	err = reader.SetLog(configLogger)
	if nil != err {
		exitwithstatus.Message("%s: new logger '%s' failed with error: %s", program, ReaderLoggerPrefix, err)
	}

	calendarLogger := logger.New(jobCalendarPrefix)
	calendar.SetLog(calendarLogger)

	// create a logger channel for the main program
	log := logger.New("main")
	defer log.Info("shutting down…")
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

	// // if requested start profiling
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

	// set the initial system mode - before any background tasks are started
	mode.Initialise(masterConfiguration.Chain)
	defer mode.Finalise()

	// ensure keys are set
	if "" == masterConfiguration.Peering.PublicKey || "" == masterConfiguration.Peering.PrivateKey {
		exitwithstatus.Message("%s: both peering Public and Private keys must be specified", program)
	}
	publicKey, err := zmqutil.ReadPublicKey(masterConfiguration.Peering.PublicKey)
	if nil != err {
		log.Criticalf("read error on: %s  error: %s", masterConfiguration.Peering.PublicKey, err)
		exitwithstatus.Message("%s: failed reading Public Key: %q  error: %s", program, masterConfiguration.Peering.PublicKey, err)
	}
	privateKey, err := zmqutil.ReadPrivateKey(masterConfiguration.Peering.PrivateKey)
	if nil != err {
		log.Criticalf("read error on: %s  error: %s", masterConfiguration.Peering.PrivateKey, err)
		exitwithstatus.Message("%s: failed reading Private Key: %q  error: %s", program, masterConfiguration.Peering.PrivateKey, err)
	}

	// general info
	log.Infof("test mode: %v", mode.IsTesting())

	// keys
	log.Tracef("public key:  %x", publicKey)
	log.Tracef("private key: %x", privateKey)

	// connection info
	log.Debugf("%s = %#v", "Peering", masterConfiguration.Peering)

	// internal queues
	ProofProxy()
	SubmitQueue()

	proofer := newProofer(logger.New(prooferLoggerPrefix), reader)
	reader.SetProofer(proofer)
	// start background processes
	// these will has blocks, changing nonce to meet difficulty
	// then submit a block to the right bitmarkd for verification
	proofer.StartHashing()

	managerLogger := logger.New(JobManagerPrefix)
	jobManager := newJobManager(
		calendar,
		proofer,
		rescheduleChannel,
		managerLogger,
	)
	jobManager.Start()

	_ = watcher.Start()

	reader.FirstTimeRun()
	reader.Start()

	p2pPrivateKey, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, 2048, rand.Reader)
	if nil != err {
		log.Errorf("generate p2p key-pair with error: %s", err)
		return
	}

	// start up bitmarkd clients these subscribe to bitmarkd
	// blocks publisher to obtain blocks for mining
connection_setup:
	for i, c := range masterConfiguration.Peering.Connect {

		slog := logger.New(fmt.Sprintf("subscriber-%d", i))

		host, err := libp2p.New(context.Background(), libp2p.Identity(p2pPrivateKey))
		if nil != err {
			slog.Errorf("new p2p host with error: %s", err)
			continue connection_setup
		}

		maddr, err := multiaddr.NewMultiaddr(c.P2P)
		if nil != err {
			slog.Errorf("new p2p maddr from %v with error: %s", c.P2P, err)
			continue connection_setup
		}

		info, err := peer.AddrInfoFromP2pAddr(maddr)
		if nil != err {
			slog.Errorf("p2p info from maddr %v with error: %s", maddr, err)
			continue connection_setup
		}

		host.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)

		s, err := host.NewStream(context.Background(), info.ID, recorderdProtocol)
		if nil != err {
			slog.Errorf("new stream with error: %s", err)
			continue connection_setup
		}

		hashRequestChan := make(chan []byte, 10)
		possibleHashChan := make(chan []byte, 10)

		err = Subscriber(i, slog, proofer, hashRequestChan)

		rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))

		mlog := logger.New(fmt.Sprintf("submitter-%d", i))
		err = Submitter(i, mlog, possibleHashChan)
		if nil != err {
			mlog.Errorf("create submitter with error: %s", err)
			continue connection_setup
		}

		go receiveMessage(rw, hashRequestChan, log)
		go sendPossibleHash(rw, possibleHashChan, log)
	}

	// erase the private key from memory
	//lint:ignore SA4006 we want to make sure we clean privateKey
	p2pPrivateKey = nil

	// abort if no clients were connected

	// wait for CTRL-C before shutting down to allow manual testing
	if 0 == len(options["quiet"]) {
		fmt.Printf("\n\nWaiting for CTRL-C (SIGINT) or 'kill <pid>' (SIGTERM)…")
	}

	// turn Signals into channel messages
	ch := make(chan os.Signal)
	//lint:ignore SA1017 XXX: signal.Notify should be buffered here
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	sig := <-ch
	log.Infof("received signal: %v", sig)
	if 0 == len(options["quiet"]) {
		fmt.Printf("\nreceived signal: %v\n", sig)
		fmt.Printf("\nshutting down...\n")
	}
}

func receiveMessage(rw *bufio.ReadWriter, hashRequestChan chan<- []byte, log *logger.L) {
	data := make([]byte, maxBytes)
	for {
		length, err := rw.Read(data)
		if nil != err {
			log.Errorf("read from stream with error: %s", err)
			continue
		}
		if length == 0 {
			return
		}
		_, fn, parameters, err := p2p.UnPackP2PMessage(data[:length])
		if nil != err {
			log.Errorf("unpack p2p message %v with error: %s", data[:length], err)
			continue
		}

		if fn == "R" {
			hashRequestChan <- parameters[0]
		} else if fn == "S" {
			_, _, parameters, err := p2p.UnPackP2PMessage(data[:length])
			if nil != err {
				log.Errorf("unpack p2p message %v with error: %s", data[:length], err)
				continue
			}
			var r interface{}
			err = json.Unmarshal(parameters[0], &r)
			if nil != err {
				log.Errorf("json unmarshal %v with error: %s", parameters[0], err)
				continue
			}
			log.Infof("receive server result: %#v", r)
		} else {
			log.Errorf("receive unknown operation: %s", fn)
		}
	}
}

func sendPossibleHash(rw *bufio.ReadWriter, possibleHashChan <-chan []byte, log *logger.L) {
	for {
		select {
		case msg := <-possibleHashChan:
			packed, err := p2p.PackP2PMessage("testing", "S", [][]byte{[]byte(msg)})
			if nil != err {
				log.Errorf("pack p2p message %v with error: %s", msg, err)
				continue
			}
			_, _ = rw.Write(packed)
			_ = rw.Flush()
		}
	}
}
