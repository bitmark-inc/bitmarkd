// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"errors"
	"fmt"
	"github.com/bitmark-inc/bitmarkd/chain"
	"github.com/bitmark-inc/bitmarkd/configuration"
	"github.com/bitmark-inc/bitmarkd/payment"
	"github.com/bitmark-inc/bitmarkd/peer"
	"github.com/bitmark-inc/bitmarkd/proof"
	"github.com/bitmark-inc/bitmarkd/publish"
	"github.com/bitmark-inc/bitmarkd/rpc"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/logger"
	"os"
	"path/filepath"
	"strings"
)

// basic defaults (directories and files are relative to the "DataDirectory" from Configuration file)
const (
	defaultDataDirectory = "" // this will error; use "." for the same directory as the config file

	defaultPeerPublicKeyFile   = "peer.private"
	defaultPeerPrivateKeyFile  = "peer.public"
	defaultProofPublicKeyFile  = "proof.private"
	defaultProofPrivateKeyFile = "proof.public"
	defaultProofSigningKeyFile = "proof.sign"
	defaultKeyFile             = "rpc.key"
	defaultCertificateFile     = "rpc.crt"

	defaultLevelDBDirectory = "data"
	defaultBitmarkDatabase  = chain.Bitmark + ".leveldb"
	defaultTestingDatabase  = chain.Testing + ".leveldb"
	defaultLocalDatabase    = chain.Local + ".leveldb"

	defaultLogDirectory = "log"
	defaultLogFile      = "bitmarkd.log"
	defaultLogCount     = 10          //  number of log files retained
	defaultLogSize      = 1024 * 1024 // rotate when <logfile> exceeds this size

	defaultRPCClients = 10
	defaultPeers      = 125
	defaultMines      = 125
)

// to hold log levels
type LoglevelMap map[string]string

// path expanded or calculated defaults
var (
	defaultLogLevels = LoglevelMap{
		logger.DefaultTag: "critical",
	}
)

type HTTPSType struct {
	MaximumConnections int      `libucl:"maximum_connections" json:"maximum_connections"`
	Listen             []string `libucl:"listen" json:"listen"`
	StatusAllowIP      []string `libucl:"status_allow_ip" json:"status_allow_ip"`
	Certificate        string   `libucl:"certificate" json:"certificate"`
	PrivateKey         string   `libucl:"private_key" json:"private_key"`
}

type DatabaseType struct {
	Directory string `libucl:"directory" json:"directory"`
	Name      string `libucl:"name" json:"name"`
}

type Configuration struct {
	DataDirectory string       `libucl:"data_directory" json:"data_directory"`
	PidFile       string       `libucl:"pidfile" json:"pidfile"`
	Chain         string       `libucl:"chain" json:"chain"`
	Nodes         string       `libucl:"nodes" json:"nodes"`
	Database      DatabaseType `libucl:"database" json:"database"`

	PeerFile          string `libucl:"peer_file" json:"peer_file"`
	ReservoirDataFile string `libucl:"reservoir_file" json:"reservoir_file"`

	ClientRPC  rpc.RPCConfiguration   `libucl:"client_rpc" json:"client_rpc"`
	HttpsRPC   rpc.HTTPSConfiguration `libucl:"https_rpc" json:"https_rpc"`
	Peering    peer.Configuration     `libucl:"peering" json:"peering"`
	Publishing publish.Configuration  `libucl:"publishing" json:"publishing"`
	Proofing   proof.Configuration    `libucl:"proofing" json:"proofing"`
	Payment    payment.Configuration  `libucl:"payment" json:"payment"`
	Logging    logger.Configuration   `libucl:"logging" json:"logging"`
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

		DataDirectory:     defaultDataDirectory,
		PidFile:           "", // no PidFile by default
		Chain:             chain.Bitmark,
		PeerFile:          "peers.json",
		ReservoirDataFile: "reservoir.json",

		Database: DatabaseType{
			Directory: defaultLevelDBDirectory,
			Name:      defaultBitmarkDatabase,
		},

		ClientRPC: rpc.RPCConfiguration{
			MaximumConnections: defaultRPCClients,
			Certificate:        defaultCertificateFile,
			PrivateKey:         defaultKeyFile,
		},

		// default: share config with normal RPC
		HttpsRPC: rpc.HTTPSConfiguration{
			MaximumConnections: defaultRPCClients,
			Certificate:        defaultCertificateFile,
			PrivateKey:         defaultKeyFile,
		},

		Peering: peer.Configuration{
			//MaximumConnections: defaultPeers,
			PublicKey:  defaultPeerPublicKeyFile,
			PrivateKey: defaultPeerPrivateKeyFile,
		},

		Publishing: publish.Configuration{
			PublicKey:  defaultPeerPublicKeyFile,
			PrivateKey: defaultPeerPrivateKeyFile,
		},

		Proofing: proof.Configuration{
			//MaximumConnections: defaultProofers,
			PublicKey:  defaultProofPublicKeyFile,
			PrivateKey: defaultProofPrivateKeyFile,
			SigningKey: defaultProofSigningKeyFile,
		},

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

	// if database was not changed from default
	if options.Database.Name == defaultBitmarkDatabase {
		switch options.Chain {
		case chain.Bitmark:
			// already correct default
		case chain.Testing:
			options.Database.Name = defaultTestingDatabase
		case chain.Local:
			options.Database.Name = defaultLocalDatabase
		default:
			return nil, errors.New(fmt.Sprintf("Chain: %s no default database setting", options.Chain))
		}
	}

	// ensure absolute data directory
	if "" == options.DataDirectory || "~" == options.DataDirectory {
		return nil, errors.New(fmt.Sprintf("Path: %q is not a valid directory", options.DataDirectory))
	} else if "." == options.DataDirectory {
		options.DataDirectory = dataDirectory // same directory as the configuration file
	} else {
		options.DataDirectory = filepath.Clean(options.DataDirectory)
	}

	// this directory must exist - i.e. must be created prior to running
	if fileInfo, err := os.Stat(options.DataDirectory); nil != err {
		return nil, err
	} else if !fileInfo.IsDir() {
		return nil, errors.New(fmt.Sprintf("Path: %q is not a directory", options.DataDirectory))
	}

	// force all relevant items to be absolute paths
	// if not, assign them to the data directory
	mustBeAbsolute := []*string{
		&options.PeerFile,
		&options.ReservoirDataFile,
		&options.Database.Directory,
		&options.ClientRPC.Certificate,
		&options.ClientRPC.PrivateKey,
		&options.HttpsRPC.Certificate,
		&options.HttpsRPC.PrivateKey,
		&options.Peering.PublicKey,
		&options.Peering.PrivateKey,
		&options.Publishing.PublicKey,
		&options.Publishing.PrivateKey,
		&options.Proofing.PublicKey,
		&options.Proofing.PrivateKey,
		&options.Proofing.SigningKey,
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
			return nil, errors.New(fmt.Sprintf("Files: %q is not plain name", *f[0]))
		}
	}

	// make absolute and create directories if they do not already exist
	for _, d := range []*string{
		&options.Database.Directory,
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
