// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/exitwithstatus"
	"github.com/bitmark-inc/getoptions"
	"github.com/syndtr/goleveldb/leveldb"
	dbutil "github.com/syndtr/goleveldb/leveldb/util"
	"golang.org/x/crypto/ssh/terminal"
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
		exitwithstatus.Message("usage: %s [--help] [--verbose] [--quiet] levedb-file table hex-prefix", program)
	}

	if len(arguments) < 3 {
		exitwithstatus.Message("%s: at least 3 arguments are required", program)
	}

	// ------------------
	// start of real main
	// ------------------

	database := arguments[0]
	table := arguments[1]
	key := arguments[2]

	if !util.EnsureFileExists(database) {
		exitwithstatus.Message("%s: missing file: %q", program, database)
	}

	if len(table) != 1 {
		exitwithstatus.Message("%s: invalid table: %q", program, table)
	}
	tableChar := table[0]

	keyBytes, err := hex.DecodeString(key)
	if err != nil {
		exitwithstatus.Message("%s: decode key error: %s", program, err)

	}

	ttyFd, err := os.OpenFile("/dev/tty", os.O_RDWR, os.ModePerm)
	if err != nil {
		exitwithstatus.Message("%s: tty open error: %s", program, err)
	}
	defer ttyFd.Close()
	oldState, err := terminal.MakeRaw(int(ttyFd.Fd()))

	if err != nil {
		exitwithstatus.Message("%s: tty open error: %s", program, err)
	}
	defer terminal.Restore(int(ttyFd.Fd()), oldState)

	console := terminal.NewTerminal(ttyFd, "DB Delete: ")

	db, err := leveldb.OpenFile(database, nil)
	if err != nil {
		exitwithstatus.Message("%s: open database: %q  error: %s", program, database, err)
	}
	defer db.Close()

	prefix := append([]byte{tableChar}, keyBytes...)
	maxRange := dbutil.Range{
		Start: prefix,                // Start of key range, included in the range
		Limit: []byte{tableChar + 1}, // Limit of key range, excluded from the range
	}

	iter := db.NewIterator(&maxRange, nil)

loop:
	for iter.Next() {

		key := iter.Key()
		value := iter.Value()

		fmt.Printf("%x → %x\r\n", key, value)
		cmd, err := console.ReadLine()
		if err != nil {
			exitwithstatus.Message("%s: terminal read error: %s", program, err)
		}
		switch strings.ToLower(cmd) {
		case "d", "y":
			db.Delete(key, nil)
		case "q":
			break loop
		default:
			fmt.Printf("invalid command\r\n")
		}
	}
	iter.Release()
	err = iter.Error()
	if err != nil {
		exitwithstatus.Message("%s: iteration error: %s", program, err)
	}

}
