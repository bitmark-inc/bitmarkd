// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"sync"
	"time"

	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/counter"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/rpc/certificate"
	"github.com/bitmark-inc/bitmarkd/rpc/handler"
	"github.com/bitmark-inc/bitmarkd/rpc/listeners"
	"github.com/bitmark-inc/bitmarkd/rpc/server"
	"github.com/bitmark-inc/logger"
)

// globals
type rpcData struct {
	sync.RWMutex // to allow locking

	log *logger.L // logger

	// set once during initialise
	initialised bool

	rpcCounter counter.Counter
}

// global data
var globalData rpcData

// Initialise - setup peer background processes
func Initialise(rpcConfiguration *listeners.RPCConfiguration, httpsConfiguration *listeners.HTTPSConfiguration, version string, ann announce.Announce) error {

	globalData.Lock()
	defer globalData.Unlock()

	// no need to start if already started
	if globalData.initialised {
		return fault.AlreadyInitialised
	}

	log := logger.New("rpc")
	globalData.log = log
	log.Info("starting…")

	tlsConfig, tlsFingerprint, err := certificate.Get(
		globalData.log,
		"rpc",
		rpcConfiguration.Certificate,
		rpcConfiguration.PrivateKey,
	)
	if nil != err {
		return err
	}

	log.Infof("rpc certificate: SHA3-256 fingerprint: %x", tlsFingerprint)

	// servers
	s := server.Create(globalData.log, version, &globalData.rpcCounter)

	rpcListener, err := listeners.NewRPC(
		rpcConfiguration,
		globalData.log,
		&globalData.rpcCounter,
		s,
		ann,
		tlsConfig,
		tlsFingerprint,
	)
	if nil != err {
		return err
	}

	err = rpcListener.Serve()
	if nil != err {
		return err
	}

	tlsConfig, _, err = certificate.Get(globalData.log, "https", httpsConfiguration.Certificate, httpsConfiguration.PrivateKey)
	if nil != err {
		return err
	}
	log.Infof("https certificate: SHA3-256 fingerprint: %x", tlsFingerprint)

	hdlr := handler.New(
		globalData.log,
		s, time.Now(),
		version,
		httpsConfiguration.MaximumConnections,
	)
	httpsListener, err := listeners.NewHTTPS(
		httpsConfiguration,
		globalData.log,
		tlsConfig,
		hdlr,
	)
	if nil != err {
		return err
	}

	err = httpsListener.Serve()
	if nil != err {
		return err
	}

	// all data initialised
	globalData.initialised = true

	return nil
}

// Finalise - stop all background tasks
func Finalise() error {

	if !globalData.initialised {
		return fault.NotInitialised
	}

	globalData.log.Info("shutting down…")
	globalData.log.Flush()

	// stop background
	//globalData.listener.Stop()

	// finally...
	globalData.initialised = false

	globalData.log.Info("finished")
	globalData.log.Flush()

	return nil
}
