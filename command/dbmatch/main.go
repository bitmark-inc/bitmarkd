// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"bytes"

	"github.com/syndtr/goleveldb/leveldb"

	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/exitwithstatus"
	"github.com/bitmark-inc/getoptions"
	"github.com/bitmark-inc/logger"
)

// set by the linker: go build -ldflags "-X main.version=M.N" ./...
var version = "zero" // do not change this value

// main program
func main() {
	// ensure exit handler is first
	defer exitwithstatus.Handler()

	flags := []getoptions.Option{
		{Long: "help", HasArg: getoptions.NO_ARGUMENT, Short: 'h'},
		{Long: "verbose", HasArg: getoptions.NO_ARGUMENT, Short: 'v'},
		{Long: "quiet", HasArg: getoptions.NO_ARGUMENT, Short: 'q'},
		{Long: "version", HasArg: getoptions.NO_ARGUMENT, Short: 'V'},
	}

	program, options, arguments, err := getoptions.GetOS(flags)
	if err != nil {
		exitwithstatus.Message("%s: getoptions error: %s", program, err)
	}

	if len(options["version"]) > 0 {
		exitwithstatus.Message("%s: version: %s", program, version)
	}

	if len(options["help"]) > 0 {
		exitwithstatus.Message("usage: %s [--help] [--verbose] [--quiet] [cmp|copy] src_db dst_db", program)
	}

	// internal logger
	logging := logger.Configuration{
		Directory: ".",
		File:      "dbmatch.log",
		Size:      1048576,
		Count:     10,
		Console:   true,
		Levels: map[string]string{
			logger.DefaultTag: "info",
		},
	}

	command := "missing arguments"

	if len(arguments) < 3 {
		exitwithstatus.Message("%s: at least 3 arguments are required", program)
	}

	// start logging
	if err = logger.Initialise(logging); err != nil {
		exitwithstatus.Message("%s: logger setup failed with error: %s", program, err)
	}
	defer logger.Finalise()

	// create a logger channel for the main program
	log := logger.New("main")
	defer log.Info("finished")
	log.Info("starting…")
	log.Infof("version: %s", version)

	// ------------------
	// start of real main
	// ------------------

	command = arguments[0]

	srcDatabase := arguments[1]
	dstDatabase := arguments[2]

	if !util.EnsureFileExists(srcDatabase) {
		exitwithstatus.Message("%s: missing file: %q", program, srcDatabase)
	}

	if !util.EnsureFileExists(dstDatabase) {
		exitwithstatus.Message("%s: missing file: %q", program, dstDatabase)
	}

	dbSrc, err := leveldb.RecoverFile(srcDatabase, nil)
	if err != nil {
		exitwithstatus.Message("%s: open src database: %q  error: %s", program, srcDatabase, err)
	}
	defer dbSrc.Close()

	dbDst, err := leveldb.RecoverFile(dstDatabase, nil)
	if err != nil {
		exitwithstatus.Message("%s: open dst database: %q  error: %s", program, dstDatabase, err)
	}
	defer dbDst.Close()

	log.Infof("src: %s", srcDatabase)
	log.Infof("dst: %s", dstDatabase)

	switch command {
	case "cmp":
		iter := dbSrc.NewIterator(nil, nil)
		totalErrors := 0
		records := 0
		log.Info("start comparison")

		for iter.Next() {

			key := iter.Key()
			value := iter.Value()

			data, err := dbDst.Get(key, nil)
			if err != nil {
				log.Errorf("read key: %x  error: %s", key, err)
				totalErrors += 1
			} else if !bytes.Equal(value, data) {
				log.Errorf("read dst key: %x  value: %x  expected: %x", key, data, value)
				totalErrors += 1
			}
			records += 1
		}
		iter.Release()
		err = iter.Error()
		if err != nil {
			exitwithstatus.Message("%s: iteration error: %s", program, err)
		}
		log.Infof("src database: %d records", records)

	case "copy":
		iter := dbSrc.NewIterator(nil, nil)
		for iter.Next() {

			key := iter.Key()
			value := iter.Value()

			err = dbDst.Put(key, value, nil)
			if err != nil {
				log.Errorf("write key: %x  error: %s", key, err)
			}

		}
		iter.Release()
		err = iter.Error()
		if err != nil {
			exitwithstatus.Message("%s: iteration error: %s", program, err)
		}
	default:
		exitwithstatus.Message("%s: unknown command: %s", program, command)

	}
}
