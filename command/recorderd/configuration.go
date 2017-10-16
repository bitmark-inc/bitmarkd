// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"errors"
	"fmt"
	"github.com/bitmark-inc/bitmarkd/chain"
	"github.com/bitmark-inc/bitmarkd/configuration"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/logger"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// basic defaults (directories and files are relative to the "DataDirectory" from Configuration file)
const (
	defaultDataDirectory = "" // this will error; use "." for the same directory as the config file

	defaultPublicKeyFile  = "prooferd.private"
	defaultPrivateKeyFile = "prooferd.public"

	defaultLogDirectory = "log"
	defaultLogFile      = "prooferd.log"
	defaultLogCount     = 10          //  number of log files retained
	defaultLogSize      = 1024 * 1024 // rotate when <logfile> exceeds this size
)

// to hold log levels
type LoglevelMap map[string]string

// path expanded or calculated defaults
var (
	defaultLogLevels = LoglevelMap{
		logger.DefaultTag: "critical",
	}
)

// server public key identification in Z85 (ZeroMQ Base-85 Encoding) see: http://rfc.zeromq.org/spec:32
type Connection struct {
	PublicKey string `libucl:"public_key" json:"public_key"`
	Blocks    string `libucl:"blocks" json:"blocks"`
	Submit    string `libucl:"submit" json:"submit"`
}

//  client keys in Z85 (ZeroMQ Base-85 Encoding) see: http://rfc.zeromq.org/spec:32
type PeerType struct {
	PrivateKey string       `libucl:"private_key" json:"private_key"`
	PublicKey  string       `libucl:"public_key" json:"public_key"`
	Connect    []Connection `libucl:"connect" json:"connect"`
}

// type PaymentType struct {
//      Account     ???? // separate private key field??
// 	Currency    string `libucl:"currency" json:"currency"`
// 	Address     string `libucl:"address" json:"address"`
// 	//Fee       string `libucl:"fee" json:"fee"` // ***** FIX THIS: can miner set its fee(s)
// }
//  add to configuration:
//	//Payment PaymentType `libucl:"payment" json:"payment"`

type Configuration struct {
	DataDirectory string               `libucl:"data_directory" json:"data_directory"`
	PidFile       string               `libucl:"pidfile" json:"pidfile"`
	Chain         string               `libucl:"chain" json:"chain"`
	Threads       int                  `libucl:"threads" json:"threads"`
	Peering       PeerType             `libucl:"peering" json:"peering"`
	Logging       logger.Configuration `libucl:"logging" json:"logging"`
}

// will read decode and verify the configuration
func getConfiguration(configurationFileName string, variables map[string]string) (*Configuration, error) {

	configurationFileName, err := filepath.Abs(filepath.Clean(configurationFileName))
	if nil != err {
		return nil, err
	}

	// absolute path to the main directory
	dataDirectory, _ := filepath.Split(configurationFileName)

	options := &Configuration{

		DataDirectory: defaultDataDirectory,
		PidFile:       "", // no PidFile by default
		Chain:         chain.Bitmark,
		Threads:       0,

		Peering: PeerType{},

		Logging: logger.Configuration{
			Directory: defaultLogDirectory,
			File:      defaultLogFile,
			Size:      defaultLogSize,
			Count:     defaultLogCount,
			Levels:    defaultLogLevels,
		},
	}

	if err := configuration.ParseConfigurationFile(configurationFileName, options, variables); err != nil {
		return nil, err
	}

	// if any test mode and the database file was not specified
	// switch to appropriate default.  Abort if then chain name is
	// not recognised.
	options.Chain = strings.ToLower(options.Chain)
	if !chain.Valid(options.Chain) {
		return nil, errors.New(fmt.Sprintf("Chain: %q is not supported", options.Chain))
	}

	// if threads invalid set number of CPUs
	if options.Threads <= 0 {
		options.Threads = runtime.NumCPU()
	}

	// ensure absolute data directory
	if "" == options.DataDirectory || "~" == options.DataDirectory {
		return nil, errors.New(fmt.Sprintf("Path: %q is not a valid directory", options.DataDirectory))
	} else if "." == options.DataDirectory {
		options.DataDirectory = dataDirectory // same directory as the configuration file
	}
	options.DataDirectory = filepath.Clean(options.DataDirectory)

	// this directory must exist - i.e. must be created prior to running
	if fileInfo, err := os.Stat(options.DataDirectory); nil != err {
		return nil, err
	} else if !fileInfo.IsDir() {
		return nil, errors.New(fmt.Sprintf("Path: %q is not a directory", options.DataDirectory))
	}

	// force all relevant items to be absolute paths
	// if not, assign them to the data directory
	mustBeAbsolute := []*string{
		&options.Peering.PublicKey,
		&options.Peering.PrivateKey,
		&options.Logging.Directory,
	}
	for _, f := range mustBeAbsolute {
		*f = util.EnsureAbsolute(options.DataDirectory, *f)
	}

	// optional absolute paths i.e. blank or an absolute path
	optionalAbsolute := []*string{
		&options.PidFile,
	}
	for _, f := range optionalAbsolute {
		if "" != *f {
			*f = util.EnsureAbsolute(options.DataDirectory, *f)
		}
	}

	// fail if any of these are not simple file names i.e. must
	// not contain path seperator, then add the correct directory
	// prefix, file item is first and corresponding directory is
	// second (or nil if no prefix can be added)
	mustNotBePaths := [][2]*string{
		{&options.Logging.File, nil},
	}
	for _, f := range mustNotBePaths {
		switch filepath.Dir(*f[0]) {
		case "", ".":
			if nil != f[1] {
				*f[0] = util.EnsureAbsolute(*f[1], *f[0])
			}
		default:
			return nil, errors.New(fmt.Sprintf("Files: %q is not plain name", *f[0]))
		}
	}

	// make absolute and create directories if they do not already exist
	for _, d := range []*string{
		&options.Logging.Directory,
	} {
		*d = util.EnsureAbsolute(options.DataDirectory, *d)
		if err := os.MkdirAll(*d, 0700); nil != err {
			return nil, err
		}
	}

	// done
	return options, nil
}
