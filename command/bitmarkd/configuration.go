// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitmark-inc/bitmarkd/chain"
	"github.com/bitmark-inc/bitmarkd/configuration"
	"github.com/bitmark-inc/bitmarkd/p2p"
	"github.com/bitmark-inc/bitmarkd/payment"
	"github.com/bitmark-inc/bitmarkd/proof"
	"github.com/bitmark-inc/bitmarkd/publish"
	"github.com/bitmark-inc/bitmarkd/rpc"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/logger"
)

// basic defaults (directories and files are relative to the "DataDirectory" from Configuration file)
const (
	defaultDataDirectory = "" // this will error; use "." for the same directory as the config file

	defaultLevelDBDirectory = "data"
	defaultBitmarkDatabase  = chain.Bitmark
	defaultTestingDatabase  = chain.Testing
	defaultLocalDatabase    = chain.Local

	defaultBitmarkCacheDirectory = chain.Bitmark + "-cache"
	defaultTestingCacheDirectory = chain.Testing + "-cache"
	defaultLocalCacheDirectory   = chain.Local + "-cache"

	defaultBitmarkBtcCacheDirectory = chain.Bitmark + "-btc-cache"
	defaultTestingBtcCacheDirectory = chain.Testing + "-btc-cache"
	defaultLocalBtcCacheDirectory   = chain.Local + "-btc-cache"

	defaultBitmarkLtcCacheDirectory = chain.Bitmark + "-ltc-cache"
	defaultTestingLtcCacheDirectory = chain.Testing + "-ltc-cache"
	defaultLocalLtcCacheDirectory   = chain.Local + "-ltc-cache"

	defaultLogDirectory = "log"
	defaultLogFile      = "bitmarkd.log"
	defaultLogCount     = 10          //  number of log files retained
	defaultLogSize      = 1024 * 1024 // rotate when <logfile> exceeds this size

	defaultRPCClients = 100          // maximum TCP connections
	defaultBandwidth  = 25 * 1000000 // 25Mbps
)

// LoglevelMap - to hold current logging levels
type LoglevelMap map[string]string

// path expanded or calculated defaults
var (
	defaultLogLevels = LoglevelMap{
		logger.DefaultTag: "critical",
	}
)

// DatabaseType - directory and name of a database
type DatabaseType struct {
	Directory string `gluamapper:"directory" json:"directory"`
	Name      string `gluamapper:"name" json:"name"`
}

// Configuration - the main configuration file data
type Configuration struct {
	DataDirectory string       `gluamapper:"data_directory" json:"data_directory"`
	PidFile       string       `gluamapper:"pidfile" json:"pidfile"`
	Chain         string       `gluamapper:"chain" json:"chain"`
	Nodes         string       `gluamapper:"nodes" json:"nodes"`
	Fastsync      bool         `gluamapper:"fast_sync" json:"fast_sync"`
	Database      DatabaseType `gluamapper:"database" json:"database"`

	CacheDirectory string                 `gluamapper:"cache_directory" json:"cache_directory"`
	DNSPeerOnly    bool                   `gluamapper:"dns_peers_only" json:"dns_peers_only"`
	ClientRPC      rpc.RPCConfiguration   `gluamapper:"client_rpc" json:"client_rpc"`
	HttpsRPC       rpc.HTTPSConfiguration `gluamapper:"https_rpc" json:"https_rpc"`
	Peering        p2p.Configuration      `gluamapper:"peering" json:"peering"`
	Publishing     publish.Configuration  `gluamapper:"publishing" json:"publishing"`
	Proofing       proof.Configuration    `gluamapper:"proofing" json:"proofing"`
	Payment        payment.Configuration  `gluamapper:"payment" json:"payment"`
	Logging        logger.Configuration   `gluamapper:"logging" json:"logging"`
}

// will read decode and verify the configuration
func getConfiguration(configurationFileName string) (*Configuration, error) {

	configurationFileName, err := filepath.Abs(filepath.Clean(configurationFileName))
	if nil != err {
		return nil, err
	}

	// absolute path to the main directory
	dataDirectory, _ := filepath.Split(configurationFileName)

	options := &Configuration{

		DataDirectory:  defaultDataDirectory,
		PidFile:        "", // no PidFile by default
		Chain:          chain.Bitmark,
		CacheDirectory: defaultBitmarkCacheDirectory,

		Database: DatabaseType{
			Directory: defaultLevelDBDirectory,
			Name:      defaultBitmarkDatabase,
		},

		ClientRPC: rpc.RPCConfiguration{
			MaximumConnections: defaultRPCClients,
			Bandwidth:          defaultBandwidth,
		},

		// default: share config with normal RPC
		HttpsRPC: rpc.HTTPSConfiguration{
			MaximumConnections: defaultRPCClients,
		},

		Payment: payment.Configuration{
			P2PCache: payment.P2PCache{
				BtcDirectory: defaultBitmarkBtcCacheDirectory,
				LtcDirectory: defaultBitmarkLtcCacheDirectory,
			},
		},

		Logging: logger.Configuration{
			Directory: defaultLogDirectory,
			File:      defaultLogFile,
			Size:      defaultLogSize,
			Count:     defaultLogCount,
			Levels:    defaultLogLevels,
		},
	}

	if err := configuration.ParseConfigurationFile(configurationFileName, options); err != nil {
		return nil, err
	}

	// if any test mode and the database file was not specified
	// switch to appropriate default.  Abort if then chain name is
	// not recognised.
	options.Chain = strings.ToLower(options.Chain)
	if !chain.Valid(options.Chain) {
		return nil, fmt.Errorf("Chain: %q is not supported", options.Chain)
	}

	// check if any option was not changed from its default above
	// if not replace with chain-specific default
	switch options.Chain {
	case chain.Bitmark:
		// already correct default
	case chain.Testing:
		if options.Database.Name == defaultBitmarkDatabase {
			options.Database.Name = defaultTestingDatabase
		}
		if options.CacheDirectory == defaultBitmarkCacheDirectory {
			options.CacheDirectory = defaultTestingCacheDirectory
		}
		if options.Payment.P2PCache.BtcDirectory == defaultBitmarkBtcCacheDirectory {
			options.Payment.P2PCache.BtcDirectory = defaultTestingBtcCacheDirectory
		}
		if options.Payment.P2PCache.LtcDirectory == defaultBitmarkLtcCacheDirectory {
			options.Payment.P2PCache.LtcDirectory = defaultTestingLtcCacheDirectory
		}
	case chain.Local:
		if options.Database.Name == defaultBitmarkDatabase {
			options.Database.Name = defaultLocalDatabase
		}
		if options.CacheDirectory == defaultBitmarkCacheDirectory {
			options.CacheDirectory = defaultLocalCacheDirectory
		}
		if options.Payment.P2PCache.BtcDirectory == defaultBitmarkBtcCacheDirectory {
			options.Payment.P2PCache.BtcDirectory = defaultLocalBtcCacheDirectory
		}
		if options.Payment.P2PCache.LtcDirectory == defaultBitmarkLtcCacheDirectory {
			options.Payment.P2PCache.LtcDirectory = defaultLocalLtcCacheDirectory
		}
	default:
		return nil, fmt.Errorf("Chain: %s no default database setting", options.Chain)
	}

	// ensure absolute data directory
	if "" == options.DataDirectory || "~" == options.DataDirectory {
		return nil, fmt.Errorf("Path: %q is not a valid directory", options.DataDirectory)
	} else if "." == options.DataDirectory {
		options.DataDirectory = dataDirectory // same directory as the configuration file
	} else {
		options.DataDirectory = filepath.Clean(options.DataDirectory)
	}

	// this directory must exist - i.e. must be created prior to running
	if fileInfo, err := os.Stat(options.DataDirectory); nil != err {
		return nil, err
	} else if !fileInfo.IsDir() {
		return nil, fmt.Errorf("Path: %q is not a directory", options.DataDirectory)
	}

	// force all relevant items to be absolute paths
	// if not, assign them to the data directory
	mustBeAbsolute := []*string{
		&options.CacheDirectory,
		&options.Database.Directory,
		&options.Payment.P2PCache.BtcDirectory,
		&options.Payment.P2PCache.LtcDirectory,
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
	// not contain path separator, then add the correct directory
	// prefix, file item is first and corresponding directory is
	// second (or nil if no prefix can be added)
	mustNotBePaths := [][2]*string{
		{&options.Database.Name, &options.Database.Directory},
		{&options.Logging.File, nil},
	}
	for _, f := range mustNotBePaths {
		switch filepath.Dir(*f[0]) {
		case "", ".":
			if nil != f[1] {
				*f[0] = util.EnsureAbsolute(*f[1], *f[0])
			}
		default:
			return nil, fmt.Errorf("Files: %q is not plain name", *f[0])
		}
	}

	// make absolute and create directories if they do not already exist
	for _, d := range []*string{
		&options.CacheDirectory,
		&options.Database.Directory,
		&options.Logging.Directory,
		&options.Payment.P2PCache.BtcDirectory,
		&options.Payment.P2PCache.LtcDirectory,
	} {
		*d = util.EnsureAbsolute(options.DataDirectory, *d)
		if err := os.MkdirAll(*d, 0700); nil != err {
			return nil, err
		}
	}

	// done
	return options, nil
}
