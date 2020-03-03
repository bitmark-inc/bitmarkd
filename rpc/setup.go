// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.e
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"sync"
	"time"

	"github.com/bitmark-inc/bitmarkd/counter"

	"github.com/bitmark-inc/bitmarkd/rpc/handler"

	"github.com/bitmark-inc/bitmarkd/rpc/listeners"

	"github.com/bitmark-inc/bitmarkd/rpc/certificate"

	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/rpc/server"
	"github.com/bitmark-inc/logger"
)

const (
	tlsName = "client_rpc"
)

// HTTPSConfiguration - configuration file data for HTTPS setup
type HTTPSConfiguration struct {
	MaximumConnections uint64              `gluamapper:"maximum_connections" json:"maximum_connections"`
	Listen             []string            `gluamapper:"listen" json:"listen"`
	Certificate        string              `gluamapper:"certificate" json:"certificate"`
	PrivateKey         string              `gluamapper:"private_key" json:"private_key"`
	Allow              map[string][]string `gluamapper:"allow" json:"allow"`
}

// globals
type rpcData struct {
	sync.RWMutex // to allow locking

	log *logger.L // logger

	// set once during initialise
	initialised bool
}

// global data
var globalData rpcData

// global atomic connection counter
// all listening ports share this count
var connectionCountRPC counter.Counter

// Initialise - setup peer background processes
func Initialise(rpcConfiguration *listeners.RPCConfiguration, httpsConfiguration *listeners.HTTPSConfiguration, version string) error {

	globalData.Lock()
	defer globalData.Unlock()

	// no need to start if already started
	if globalData.initialised {
		return fault.AlreadyInitialised
	}

	log := logger.New("rpc")
	globalData.log = log
	log.Info("starting…")

	tlsConfig, certificateFingerprint, err := certificate.Get(globalData.log, tlsName, rpcConfiguration.Certificate, rpcConfiguration.PrivateKey)
	if nil != err {
		return err
	}
	log.Infof("%s: SHA3-256 fingerprint: %x", tlsName, certificateFingerprint)

	// servers
	s := server.Create(log, version, &connectionCountRPC)
	rpcListener, err := listeners.NewRPC(
		rpcConfiguration,
		log,
		&connectionCountRPC,
		s,
		announce.Get(),
		tlsConfig,
		certificateFingerprint,
	)
	if nil != err {
		return err
	}
	err = rpcListener.Serve()
	if nil != err {
		return err
	}

	h := handler.New(
		log,
		s,
		time.Now(),
		version,
		httpsConfiguration.MaximumConnections,
	)

	httpsListener, err := listeners.NewHTTPS(
		httpsConfiguration,
		log,
		tlsConfig,
		h,
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

	// finally...
	globalData.initialised = false

	globalData.log.Info("finished")
	globalData.log.Flush()

	return nil
}
