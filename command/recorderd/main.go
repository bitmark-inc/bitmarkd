// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	// initialise encryption
	//err = zmqutil.StartAuthentication()
	//if nil != err {
	//	log.Criticalf("zmq.AuthStart(): error: %s", err)
	//	exitwithstatus.Message("%s: zmq.AuthStart() error: %s", program, err)
	//}
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

	r := rand.Reader
	p2pPrivateKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if nil != err {
		panic(err)
	}

	// start up bitmarkd clients these subscribe to bitmarkd
	// blocks publisher to obtain blocks for mining
	//connection_setup:
	for _, c := range masterConfiguration.Peering.Connect {

		//serverPublicKey, err := zmqutil.ReadPublicKey(remote.PublicKey)
		//if nil != err {
		//	log.Warnf("client: %d invalid server publickey: %q error: %s", i, remote.PublicKey, err)
		//	continue connection_setup
		//}
		//
		//bc, err := util.NewConnection(remote.Blocks)
		//if nil != err {
		//	log.Warnf("client: %d invalid blocks publisher: %q error: %s", i, remote.Blocks, err)
		//	continue connection_setup
		//}
		//blocksAddress, blocksv6 := bc.CanonicalIPandPort("tcp://")
		//
		//sc, err := util.NewConnection(remote.Submit)
		//if nil != err {
		//	log.Warnf("client: %d invalid submit address: %q error: %s", i, remote.Submit, err)
		//	continue connection_setup
		//}
		//submitAddress, submitv6 := sc.CanonicalIPandPort("tcp://")
		//
		//log.Infof("client: %d subscribe: %q  submit: %q", i, remote.Blocks, remote.Submit)
		//
		//mlog := logger.New(fmt.Sprintf("submitter-%d", i))
		// TODO: libp2p
		//err = Submitter(i, submitAddress, submitv6, serverPublicKey, publicKey, privateKey, mlog)
		//if nil != err {
		//	log.Warnf("submitter: %d failed error: %s", i, err)
		//	continue connection_setup
		//}

		//host.SetStreamHandler("/recorderd/1.0.0", SubmitterHandler)
		//var port string
		//for _, la := range host.Network().ListenAddresses() {
		//	if p, err := la.ValueForProtocol(multiaddr.P_TCP); nil == err {
		//		port = p
		//		break
		//	}
		//}
		//if "" == port {
		//	panic("was not able to find actual local port")
		//}

		//slog := logger.New(fmt.Sprintf("subscriber-%d", i))
		// TODO: libp2p
		//err = Subscribe(i, blocksAddress, blocksv6, serverPublicKey, publicKey, privateKey, slog, proofer)
		//if nil != err {
		//	log.Warnf("subscribe: %d failed error: %s", i, err)
		//	continue connection_setup
		//}

		//r := rand.Reader
		//prvKey, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, 2048, r)
		//prvKeyBytes, _ := prvKey.Bytes()

		//marshalKey, err := crypto.MarshalPrivateKey(prvKey)
		//if err != nil {
		//	fmt.Printf("marshal private key with error: %s\n", err)
		//}
		//hexEncodeKey := make([]byte, hex.EncodedLen(len(marshalKey)))
		//hex.Encode(hexEncodeKey, marshalKey)

		host, err := libp2p.New(context.Background(), libp2p.Identity(p2pPrivateKey))
		if nil != err {
			panic(err)
		}

		fmt.Printf("p2p: %s\n", c.P2P)
		maddr, err := multiaddr.NewMultiaddr(c.P2P)
		if nil != err {
			panic(err)
		}

		info, err := peer.AddrInfoFromP2pAddr(maddr)
		if nil != err {
			panic(err)
		}
		fmt.Printf("info: %#v\n", info)

		host.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)

		s, err := host.NewStream(context.Background(), info.ID, "/recorderd/1.0.0")
		if nil != err {
			panic(err)
		}

		rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))

		go readData(rw)
		go writeData(rw)
	}

	// erase the private key from memory
	//lint:ignore SA4006 we want to make sure we clean privateKey
	//privateKey = []byte{}

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

func readData(rw *bufio.ReadWriter) {
	maxBytes := 1000
	data := make([]byte, maxBytes)
	for {
		length, err := rw.Read(data)
		if nil != err {
			panic(err)
		}
		if length == 0 {
			return
		}
		chain, fn, parameters, err := p2p.UnPackP2PMessage(data[:length])
		if nil != err {
			panic(err)
		}
		fmt.Printf("received chain: %s, fn: %s, parameter: %s\n", chain, fn, string(parameters[0]))
	}
}

func writeData(rw *bufio.ReadWriter) {
	for {
		select {
		case <-time.After(8 * time.Second):
			str := fmt.Sprintf("%s\n", time.Now())
			packed, err := p2p.PackP2PMessage("testing", "R", [][]byte{[]byte(str)})
			if nil != err {
				panic(err)
			}
			rw.Write(packed)
			rw.Flush()
		}
	}
}
