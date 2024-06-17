// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// Copyright (c) 2014-2020 Bitmark Inc.
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
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/logger"
)

// basic defaults (directories and files are relative to the "DataDirectory" from Configuration file)
const (
	defaultDataDirectory = "" // this will error; use "." for the same directory as the config file

	defaultPublicKeyFile  = "simbm.private"
	defaultPrivateKeyFile = "simbm.public"

	defaultLogDirectory = "log"
	defaultLogFile      = "simbm.log"
	defaultLogCount     = 10          //  number of log files retained
	defaultLogSize      = 1024 * 1024 // rotate when <logfile> exceeds this size
)

// LoglevelMap - to hold log levels
type LoglevelMap map[string]string

// path expanded or calculated defaults
var (
	defaultLogLevels = LoglevelMap{
		"main":            "info",
		logger.DefaultTag: "critical",
	}
)

// server identification in Z85 (ZeroMQ Base-85 Encoding) see: http://rfc.zeromq.org/spec:32
type ProoferType struct {
	PrivateKey string   `gluamapper:"private_key" json:"private_key"`
	PublicKey  string   `gluamapper:"public_key" json:"public_key"`
	Publish    []string `gluamapper:"publish" json:"publish"`
	Submit     []string `gluamapper:"submit" json:"submit"`
}

// Configuration - configuration file data
type Configuration struct {
	DataDirectory string               `gluamapper:"data_directory" json:"data_directory"`
	PidFile       string               `gluamapper:"pidfile" json:"pidfile"`
	Chain         string               `gluamapper:"chain" json:"chain"`
	Proofer       ProoferType          `gluamapper:"proofer" json:"proofer"`
	Logging       logger.Configuration `gluamapper:"logging" json:"logging"`
}

// will read decode and verify the configuration
func getConfiguration(configurationFileName string) (*Configuration, error) {

	configurationFileName, err := filepath.Abs(filepath.Clean(configurationFileName))
	if err != nil {
		return nil, err
	}

	// absolute path to the main directory
	dataDirectory, _ := filepath.Split(configurationFileName)

	options := &Configuration{

		DataDirectory: defaultDataDirectory,
		PidFile:       "", // no PidFile by default
		Chain:         chain.Bitmark,
		Proofer:       ProoferType{},
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

	// ensure absolute data directory
	if options.DataDirectory == "" || options.DataDirectory == "~" {
		return nil, fmt.Errorf("Path: %q is not a valid directory", options.DataDirectory)
	} else if options.DataDirectory == "." {
		options.DataDirectory = dataDirectory // same directory as the configuration file
	}
	options.DataDirectory = filepath.Clean(options.DataDirectory)

	// this directory must exist - i.e. must be created prior to running
	if fileInfo, err := os.Stat(options.DataDirectory); err != nil {
		return nil, err
	} else if !fileInfo.IsDir() {
		return nil, fmt.Errorf("Path: %q is not a directory", options.DataDirectory)
	}

	// force all relevant items to be absolute paths
	// if not, assign them to the data directory
	mustBeAbsolute := []*string{
		&options.Proofer.PublicKey,
		&options.Proofer.PrivateKey,
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
		if *f != "" {
			*f = util.EnsureAbsolute(options.DataDirectory, *f)
		}
	}

	// fail if any of these are not simple file names i.e. must
	// not contain path seperator then add the correct directory
	// prefix, file item is first and corresponding directory is
	// second (or nil if no prefix can be added)
	mustNotBePaths := [][2]*string{
		{&options.Logging.File, nil},
	}
	for _, f := range mustNotBePaths {
		switch filepath.Dir(*f[0]) {
		case "", ".":
			if f[1] != nil {
				*f[0] = util.EnsureAbsolute(*f[1], *f[0])
			}
		default:
			return nil, fmt.Errorf("Files: %q is not plain name", *f[0])
		}
	}

	// make absolute and create directories if they do not already exist
	for _, d := range []*string{
		&options.Logging.Directory,
	} {
		*d = util.EnsureAbsolute(options.DataDirectory, *d)
		if err := os.MkdirAll(*d, 0o700); err != nil {
			return nil, err
		}
	}

	// done
	return options, nil
}
