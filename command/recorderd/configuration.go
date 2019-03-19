// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitmark-inc/bitmarkd/chain"
	"github.com/bitmark-inc/bitmarkd/configuration"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/logger"
)

// basic defaults (directories and files are relative to the "DataDirectory" from Configuration file)
const (
	defaultDataDirectory = "" // this will error; use "." for the same directory as the config file

	defaultLogDirectory = "log"
	defaultLogFile      = "recorderd.log"
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
	PublicKey string `gluamapper:"public_key" json:"public_key"`
	Blocks    string `gluamapper:"blocks" json:"blocks"`
	Submit    string `gluamapper:"submit" json:"submit"`
}

type PeerType struct {
	//  client keys in Z85 (ZeroMQ Base-85 Encoding) see: http://rfc.zeromq.org/spec:32
	PrivateKey string       `gluamapper:"private_key" json:"private_key"`
	PublicKey  string       `gluamapper:"public_key" json:"public_key"`
	Connect    []Connection `gluamapper:"connect" json:"connect"`
}

type ConfigCalendar struct {
	Monday    string `gluamapper:"monday" json:"monday"`
	Tuesday   string `gluamapper:"tuesday" json:"tuesday"`
	Wednesday string `gluamapper:"wednesday" json:"wednesday"`
	Thursday  string `gluamapper:"thursday" json:"thursday"`
	Friday    string `gluamapper:"friday" json:"friday"`
	Saturday  string `gluamapper:"saturday" json:"saturday"`
	Sunday    string `gluamapper:"sunday" json:"sunday"`
}

// type PaymentType struct {
//      Account     ???? // separate private key field??
// 	Currency    string `gluamapper:"currency" json:"currency"`
// 	Address     string `gluamapper:"address" json:"address"`
// 	//Fee       string `gluamapper:"fee" json:"fee"` // ***** FIX THIS: can miner set its fee(s)
// }
//  add to configuration:
//	//Payment PaymentType `gluamapper:"payment" json:"payment"`

type Configuration struct {
	DataDirectory string               `gluamapper:"data_directory" json:"data_directory"`
	PidFile       string               `gluamapper:"pidfile" json:"pidfile"`
	Chain         string               `gluamapper:"chain" json:"chain"`
	MaxCPUUsage   int                  `gluamapper:"max_cpu_usage" json:"max_cpu_usage`
	Calendar      ConfigCalendar       `gluamapper:"calendar" json:"calendar"`
	Peering       PeerType             `gluamapper:"peering" json:"peering"`
	Logging       logger.Configuration `gluamapper:"logging" json:"logging"`
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

		DataDirectory: defaultDataDirectory,
		PidFile:       "", // no PidFile by default
		Chain:         chain.Bitmark,
		MaxCPUUsage:   50,
		Calendar:      ConfigCalendar{},

		Peering: PeerType{},

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
		return nil, errors.New(fmt.Sprintf("Chain: %q is not supported", options.Chain))
	}

	if options.MaxCPUUsage <= 0 || options.MaxCPUUsage > 100 {
		options.MaxCPUUsage = 50
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

func (c *Configuration) maxCPUUsage() int {
	return c.MaxCPUUsage
}
