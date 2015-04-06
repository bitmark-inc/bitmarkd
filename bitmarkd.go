// Copyright (c) 2014-2015 Bitmark Inc.
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
	"github.com/bitmark-inc/exitwithstatus"
	"github.com/bitmark-inc/listener"
	"github.com/bitmark-inc/logger"
	"os"
	"os/signal"
	"syscall"
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

// to check if PID file was created
var lockWasCreated = false

func main() {
	// ensure exit handler is first
	defer exitwithstatus.Handler()
	defer fmt.Printf("\nprogram exit\n")
	defer logger.Finalise()

	// read options and parse the configuration file
	// also sets up and starts logging
	options := configuration.ParseOptions()

	if options.Version {
		exitwithstatus.Usage("Version: %s\n", Version())
	}

	// start logging
	err := logger.Initialise(options.LogFile, options.LogSize, options.LogRotateCount)
	if err != nil {
		exitwithstatus.Usage("Logger setup failed with error: %v\n", err)
	}
	logger.LoadLevels(options.Debug)

	// create a logger channel for the main program
	log := logger.New("main")
	defer log.Info("shutting down…")
	log.Info("starting…")
	log.Debugf("options: %v", options)

	// set up the fault panic log (now that logging is available
	fault.Initialise()
	defer fault.Finalise()

	// ------------------
	// start of real main
	// ------------------

	// grab lock file or fail
	lf, err := os.OpenFile(options.PidFile, os.O_WRONLY|os.O_EXCL|os.O_CREATE, os.ModeExclusive|0600)
	if err != nil {
		if os.IsExist(err) {
			exitwithstatus.Usage("Another instance is already running\n")
		}
		exitwithstatus.Usage("PID file: %s creation failed with error: %v\n", options.PidFile, err)
	}
	fmt.Fprintf(lf, "%d\n", os.Getpid())
	lf.Close()
	lockWasCreated = true
	defer removeAppLock(options.PidFile)

	// command processing - need lock so do not affect an already running process
	if "" != options.Args.Command {
		processCommand(log, options)
		return
	}

	// set the initial system mode - before any background tasks are started
	mode.Initialise()
	defer mode.Finalise()
	mode.SetTesting(options.TestMode)

	// ensure keys are set
	if "" == options.PublicKey || "" == options.PrivateKey {
		exitwithstatus.Usage("Both Public and Private keys must be specified\n")
	}
	publicKey, err := readKeyFile(options.PublicKey)
	if nil != err {
		log.Criticalf("read Public Key error = %v", err)
		exitwithstatus.Usage("read Public Key error = %v\n", err)
	}
	privateKey, err := readKeyFile(options.PrivateKey)
	if nil != err {
		log.Criticalf("read Private Key error = %v", err)
		exitwithstatus.Usage("read Private Key error = %v\n", err)
	}

	// server identification
	log.Debugf("%s = '%v'", "PublicKey", options.PublicKey)
	log.Debugf("%s = '%v'", "PrivateKey", options.PrivateKey)

	// info abount mode
	log.Infof("test mode: %v", mode.IsTesting())
	log.Infof("database: %s", options.DatabaseFile)

	// RPC
	log.Debugf("%s = '%v'", "RPCClients", options.RPCClients)
	log.Debugf("%s = '%v'", "RPCListeners", options.RPCListeners)
	log.Debugf("%s = '%v'", "RPCCertificate", options.RPCCertificate)
	log.Debugf("%s = '%v'", "RPCKey", options.RPCKey)

	// peer
	log.Debugf("%s = '%v'", "Peers", options.Peers)
	log.Debugf("%s = '%v'", "PeerListeners", options.PeerListeners)
	//log.Debugf("%s = '%v'", "PeerCertificate", options.PeerCertificate)
	//log.Debugf("%s = '%v'", "PeerKey", options.PeerKey)

	// start the memory pool
	log.Info("start pool")
	pool.Initialise(options.DatabaseFile)
	defer pool.Finalise()

	// block data storage - depends on pool
	log.Info("initialise block")
	block.Initialise(options.BlockCacheSize)
	defer block.Finalise()

	// transaction data storage - depends on pool
	log.Info("initialise transaction")
	transaction.Initialise(options.TransactionCacheSize)
	defer transaction.Finalise()

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
			limit:               options.RPCClients,
			addresses:           options.RPCListeners,
			certificateFileName: options.RPCCertificate,
			keyFileName:         options.RPCKey,
			callback:            rpc.Callback,
			argument: &rpc.ServerArgument{
				Log: rpcLog,
			},
		},
		// "peer": {
		// 	limit:               options.Peers,
		// 	addresses:           options.PeerListeners,
		// 	certificateFileName: options.PeerCertificate,
		// 	keyFileName:         options.PeerKey,
		// 	callback:            p2p.Callback,
		// 	argument: &p2p.ServerArgument{
		// 		Log: peerLog,
		// 	},
		// },
		"mine": {
			limit:               options.Mines,
			addresses:           options.MineListeners,
			certificateFileName: options.MineCertificate,
			keyFileName:         options.MineKey,
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
			for _, address := range options.RPCAnnounce {
				announce.AddPeer(address, announce.TypeRPC, &peerData)
			}
		case "peer":
			for _, address := range options.PeerAnnounce {
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

	// // read static certificates for remotes
	// certificatePool := x509.NewCertPool()
	// for i, certificateFileName := range options.RemoteCertificate {
	// 	path, exists := resolveFileName(certificateFileName)
	// 	if !exists {
	// 		log.Errorf("certificate: %d: does not exist: in '%s' or '%s'", i, certificateFileName, path)
	// 		continue
	// 	}
	// 	certificatePEM, err := ioutil.ReadFile(path)
	// 	if err != nil {
	// 		log.Errorf("failed to read certificate: %d: '%s' error: %v", i, path, err)
	// 		continue
	// 	}
	// 	var certificateDER *pem.Block
	// 	ok := false
	// 	for {
	// 		certificateDER, certificatePEM = pem.Decode(certificatePEM)
	// 		if nil == certificateDER {
	// 			break
	// 		}
	// 		if "CERTIFICATE" == certificateDER.Type {
	// 			ok = false // ensure fail even if earlier certificates wer detected
	// 			certificate, err := x509.ParseCertificate(certificateDER.Bytes)
	// 			if err != nil {
	// 				continue
	// 			}

	// 			certificatePool.AddCert(certificate)

	// 			fingerprint := util.Fingerprint(certificate.Raw)
	// 			log.Infof("remote fingerprint = %x", fingerprint)
	// 			// store certificate
	// 			announce.AddCertificate(&fingerprint, certificate.Raw)
	// 			ok = true
	// 		}
	// 	}

	// 	if !ok {
	// 		log.Errorf("certificate: %d bad certificate in file: '%s'", i, certificateFileName)
	// 	}
	// }

	// // P2P system start
	// log.Infof("start p2p, maximum outgoing = %d", options.Remotes)
	// if err := p2p.Initialise(options.Remotes, certificatePool, myFingerprints); nil != err {
	// 	log.Criticalf("p2p start error: %v", err)
	// 	exitwithstatus.Exit(1)
	// }
	// defer p2p.Finalise()

	// connect to various payment services
	err = payment.BitcoinInitialise(options.BitcoinURL, options.BitcoinUsername, options.BitcoinPassword, options.BitcoinAddress, options.BitcoinFee, options.BitcoinStart)
	if nil != err {
		log.Criticalf("failed to initialise Bitcoin  error: %v", err)
		exitwithstatus.Exit(1)
	}

	// start up the peering
	err = peer.Initialise(options.PeerListeners, publicKey, privateKey)
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
	for i, remote := range options.RemoteConnect {
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
	if !options.Quiet {
		fmt.Printf("\n\nWaiting for CTRL-C (SIGINT) or 'kill <pid>' (SIGTERM)…")
	}

	// turn Signals into channel messages
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	sig := <-ch
	log.Infof("received signal: %v", sig)
	if !options.Quiet {
		fmt.Printf("\nreceived signal: %v\n", sig)
		fmt.Printf("\nshutting down...\n")
	}
}

// remove the lock file - only if this instance created it
func removeAppLock(appLockFile string) {
	if lockWasCreated {
		os.Remove(appLockFile)
		lockWasCreated = false
	}
}
