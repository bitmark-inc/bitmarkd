// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package bitcoin

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/fault"
	// "github.com/bitmark-inc/bitmarkd/payment"
	"github.com/bitmark-inc/logger"
	"io/ioutil"
	"net/http"
	"sync"
	// "time"
)

// global constants
const (
	bitcoinMinimumVersion = 90200 // do not start if bitcoind older than this
	// bitcoinRateLimit      = 15.0            // blocks/second
	// bitcoinPollingTime    = 2 * time.Minute // sample bitcoin "blockcount" RPC at this interval
	// bitcoinMaximumRetries = 10              // panic after this many consecutive errors
	// bitcoinCurrencyName   = "bitcoin"       // all lowercase currency string
	// bitcoinBlockRange     = 200             // number of blocks to consider as relevant
	// bitcoinConfirmations  = 3               // stop processing this many blocks back from most recent block
	//
	// // this is how far back in the bitcoin block chain to start when process begins
	// bitcoinBlockOffset = bitcoinBlockRange + bitcoinConfirmations
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

	// queueing
	blockQueue chan uint64
	itemQueue  chan *priorityItem

	// identifier for the RPC
	id uint64

	// // payment info
	// minerAddress      string
	// feeSingle         uint64 // value in Satoshis avoid float because of rounding errors
	// feeMultiple       uint64 // value in Satoshis avoid float because of rounding errors
	latestBlockNumber uint64 // value from bitcoind

	// for background
	background *background.T

	// set once during initialise
	initialised bool
}

// global data
var globalData bitcoinData

// external API
// ------------

// a block of configuration data
// this is read from a libucl configuration file
type Configuration struct {
	Username      string `libucl:"username"`
	Password      string `libucl:"password"`
	URL           string `libucl:"url"`
	CACertificate string `libucl:"ca_certificate"`
	Certificate   string `libucl:"certificate"`
	PrivateKey    string `libucl:"private_key"`
	// Address       string `libucl:"address"`
	// Fee           string `libucl:"fee"`
}

// initialise for bitcoin payments
// also calls the internal initialisePayment() and register()
//
// Note fee is a string value and is converted to Satoshis to avoid rounding errors
func Initialise(configuration Configuration) error {

	// // ensure payments are initialised
	// if err := payment.Initialise(); nil != err {
	// 	return err
	// }

	globalData.Lock()
	defer globalData.Unlock()

	// no need to start if already started
	if globalData.initialised {
		return fault.ErrAlreadyInitialised
	}

	// // ***** FIX THIS: what about the payment for this miner?
	// if "" == configuration.Address {
	// 	return fault.ErrPaymentAddressMissing
	// }

	globalData.log = logger.New("bitcoin")
	if nil == globalData.log {
		return fault.ErrInvalidLoggerChannel
	}
	globalData.log.Info("starting…")

	globalData.id = 0
	globalData.username = configuration.Username
	globalData.password = configuration.Password
	globalData.url = configuration.URL

	// set up queues
	globalData.blockQueue = make(chan uint64, 10)
	globalData.itemQueue = make(chan *priorityItem, 10)

	// globalData.minerAddress = configuration.Address
	// globalData.feeSingle = convertToSatoshi([]byte(configuration.Fees.Single))
	// globalData.feeMultiple = convertToSatoshi([]byte(configuration.Fees.Multiple))

	if "" != configuration.Certificate {
		keyPair, err := tls.LoadX509KeyPair(configuration.Certificate, configuration.PrivateKey)
		if nil != err {
			return err
		}

		certificatePool := x509.NewCertPool()

		data, err := ioutil.ReadFile(configuration.CACertificate)
		if err != nil {
			globalData.log.Criticalf("failed to parse certificate from: %q", configuration.CACertificate)
			return err
		}

		if !certificatePool.AppendCertsFromPEM(data) {
			globalData.log.Criticalf("failed to parse certificate from: %q", configuration.CACertificate)
			return err
		}

		tlsConfiguration := &tls.Config{
			Certificates:             []tls.Certificate{keyPair},
			RootCAs:                  certificatePool,
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
		globalData.client = &http.Client{}
	}

	// all data initialised
	globalData.initialised = true

	globalData.log.Debug("getinfo…")

	// query bitcoind for status
	// only need to have necessary fields as JSON unmarshaller will igtnore excess
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
	}

	// set up current block number
	globalData.latestBlockNumber = reply.Blocks
	globalData.log.Debugf("block count: %d", globalData.latestBlockNumber)

	// start background processes
	globalData.log.Info("start background…")

	// list of background processes to start
	var processes = background.Processes{
		&globalData,
	}

	globalData.background = background.Start(processes, globalData.log)

	// payment.Register(bitcoinCurrencyName, &payment.CallType{
	// 	Miner: getAddress,
	// })

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

	// // finalise the main subsystem
	// return payment.Finalise()

	return nil
}

// // for mining
// // ----------

// // to get the current address as string for mining
// func getAddress() string {
// 	globalData.RLock()
// 	defer globalData.RUnlock()

// 	return globalData.minerAddress
// }
