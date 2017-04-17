// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package bitcoin

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/logger"
	"io/ioutil"
	"net/http"
	"sync"
)

// global constants
const (
	bitcoinMinimumVersion = 120100 // do not start if bitcoind olde
)

// globals for background proccess
type bitcoinData struct {
	sync.RWMutex // to allow locking

	// logger
	log *logger.L

	// connection to bitcoin daemon
	client *http.Client
	url    string

	// authentication
	username string
	password string

	// identifier for the RPC
	id uint64

	// values from bitcoind
	latestBlockNumber uint64
	latestBlockHash   string

	// to reduce the number of Currency record overwrites
	saveCount uint64

	// for background
	background *background.T

	// set once during initialise
	initialised bool
}

// global data
var globalData bitcoinData

// a block of configuration data
// this is read from a libucl configuration file
type Configuration struct {
	Username        string `libucl:"username"`
	Password        string `libucl:"password"`
	URL             string `libucl:"url"`
	ServerName      string `libucl:"server_name"`
	CACertificate   string `libucl:"ca_certificate"`
	Certificate     string `libucl:"certificate"`
	PrivateKey      string `libucl:"private_key"`
	Block           uint64 `libucl:"block"`
	Hash            string `libucl:"hash"`
	ResetBlockCount bool   `libucl:"reset_block_count"`
}

// initialise for bitcoin payments
// also calls the internal initialisePayment() and register()
func Initialise(configuration *Configuration) error {

	globalData.Lock()
	defer globalData.Unlock()

	// no need to start if already started
	if globalData.initialised {
		return fault.ErrAlreadyInitialised
	}

	globalData.log = logger.New("bitcoin")
	if nil == globalData.log {
		return fault.ErrInvalidLoggerChannel
	}
	globalData.log.Info("starting…")

	globalData.id = 0
	globalData.username = configuration.Username
	globalData.password = configuration.Password
	globalData.url = configuration.URL

	useTLS := false
	clientCertificates := []tls.Certificate(nil)

	if "" != configuration.Certificate {
		keyPair, err := tls.LoadX509KeyPair(configuration.Certificate, configuration.PrivateKey)
		if nil != err {
			globalData.log.Criticalf("parse certificate: %q  private key: %q  error: %v", configuration.Certificate, configuration.PrivateKey, err)
			return err
		}
		clientCertificates = []tls.Certificate{keyPair}
		useTLS = true
	}

	certificatePool := x509.NewCertPool()
	if "" != configuration.CACertificate {
		data, err := ioutil.ReadFile(configuration.CACertificate)
		if err != nil {
			globalData.log.Criticalf("parse CA certificate from: %q  error: %v", configuration.CACertificate, err)
			return err
		}

		if !certificatePool.AppendCertsFromPEM(data) {
			globalData.log.Criticalf("pool certificate from: %q  error: %v", configuration.CACertificate, err)
			return err
		}
		useTLS = true
	}

	if useTLS {
		// use TLS in one of two cases:
		// a) only CA certificate is provided
		// b) all three: clients certificate and private key, plus CA certificate
		// server name is the name embedded in the certificate
		tlsConfiguration := &tls.Config{
			Certificates:             clientCertificates,
			RootCAs:                  certificatePool,
			NextProtos:               nil,
			ServerName:               configuration.ServerName, // the server name in the certificate
			InsecureSkipVerify:       false,
			CipherSuites:             nil,
			PreferServerCipherSuites: true,
			MinVersion:               12, // force 1.2 and above
			MaxVersion:               0,  // no maximum
			CurvePreferences:         nil,
		}

		globalData.client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsConfiguration,
			},
		}
	} else {
		// plain http
		globalData.client = &http.Client{}
	}

	// all data initialised
	globalData.initialised = true

	globalData.log.Debug("getinfo…")

	// query bitcoind for status
	// only need to have necessary fields as JSON unmarshaller will ignore excess
	var reply struct {
		Version uint64 `json:"version"`
		Blocks  uint64 `json:"blocks"`
	}
	err := bitcoinCall("getinfo", []interface{}{}, &reply)
	if nil != err {
		return err
	}

	// check version is sufficient
	if reply.Version < bitcoinMinimumVersion {
		globalData.log.Errorf("Bitcoin version: %d < allowed: %d", reply.Version, bitcoinMinimumVersion)
		return fault.ErrInvalidVersion
	} else {
		globalData.log.Infof("Bitcoin version: %d", reply.Version)
		globalData.log.Infof("Bitcoin block height: %d", reply.Blocks)
	}

	// set up current block number
	globalData.latestBlockNumber = 1
	globalData.latestBlockHash = ""
	globalData.saveCount = 0

	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, currency.Bitcoin.Uint64())
	if configuration.ResetBlockCount {
		globalData.log.Warnf("resetting the bitcoin block database starting from block: %d", configuration.Block)
		var hash string
		err := bitcoinGetBlockHash(configuration.Block, &hash)
		if nil != err {
			return err
		}
		if configuration.Hash != hash {
			globalData.log.Criticalf("returned hash: %s  but expected hash: %s", hash, configuration.Hash)
			globalData.log.Critical("check configuration section: bitcoin")
			return fault.ErrInitialisationFailed
		}
		globalData.log.Warnf("saving block: %d  hash: %s", configuration.Block, hash)
		saveBlockCount(configuration.Block, hash)
		globalData.log.Warn("SUGGESTION: change reset_block_count to false in configuration file")
		globalData.log.Warn("SUGGESTION: as this will speed up net start")
	}
	record := storage.Pool.Currency.Get(key)
	if nil != record {
		globalData.latestBlockNumber = binary.BigEndian.Uint64(record[:8])
		globalData.latestBlockHash = string(record[8:])
		globalData.log.Infof("latest block on file:: %d", globalData.latestBlockNumber)
		globalData.log.Infof("latest block hash: %s", globalData.latestBlockHash)
	}

	// start background processes
	globalData.log.Info("start background…")

	// list of background processes to start
	var processes = background.Processes{
		&globalData,
	}

	globalData.background = background.Start(processes, globalData.log)

	return nil
}

// finialise - stop all background tasks
// also calls the internal finalisePayment()
func Finalise() error {
	globalData.Lock()
	defer globalData.Unlock()

	if !globalData.initialised {
		return fault.ErrNotInitialised
	}

	globalData.log.Info("shutting down…")
	globalData.log.Flush()

	// stop background
	globalData.background.Stop()

	// finally...
	globalData.initialised = false

	globalData.log.Info("finished")
	globalData.log.Flush()

	return nil
}
