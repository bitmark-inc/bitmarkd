// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/exitwithstatus"
	"github.com/bitmark-inc/getoptions"
	"github.com/bitmark-inc/logger"
)

// set by the linker: go build -ldflags "-X main.version=M.N" ./...
var version = "zero" // do not change this value

// colours
const (
	keyColour1  = "\033[1;36m"
	keyColour2  = "\033[1;31m"
	valColour1  = "\033[1;33m"
	valColour2  = "\033[1;34m"
	delColour1  = "\033[1;35m"
	delColour2  = "\033[0;35m"
	delColour3  = "\033[0;31m"
	delColour4  = "\033[1;35m"
	nodelColour = "\033[1;32m"
	endColour   = "\033[0m"
)

// main program
func main() {
	// ensure exit handler is first
	defer exitwithstatus.Handler()

	flags := []getoptions.Option{
		{Long: "help", HasArg: getoptions.NO_ARGUMENT, Short: 'h'},
		{Long: "verbose", HasArg: getoptions.NO_ARGUMENT, Short: 'v'},
		{Long: "version", HasArg: getoptions.NO_ARGUMENT, Short: 'V'},
		{Long: "list", HasArg: getoptions.NO_ARGUMENT, Short: 'l'},
		{Long: "delete", HasArg: getoptions.NO_ARGUMENT, Short: 'd'},
		{Long: "early", HasArg: getoptions.NO_ARGUMENT, Short: 'e'},
		{Long: "colour", HasArg: getoptions.NO_ARGUMENT, Short: 'g'},
		{Long: "ascii", HasArg: getoptions.NO_ARGUMENT, Short: 'a'},
		{Long: "file", HasArg: getoptions.REQUIRED_ARGUMENT, Short: 'f'},
		{Long: "count", HasArg: getoptions.REQUIRED_ARGUMENT, Short: 'c'},
	}

	program, options, arguments, err := getoptions.GetOS(flags)
	if nil != err {
		exitwithstatus.Message("%s: getoptions error: %s", program, err)
	}

	if len(options["version"]) > 0 {
		exitwithstatus.Message("%s: version: %s", program, version)
	}

	if len(options["list"]) > 0 {

		// this will be a struct type
		poolType := reflect.TypeOf(storage.Pool)

		// print all available tags
		fmt.Printf(" tags:\n")
		for i := 0; i < poolType.NumField(); i += 1 {
			fieldInfo := poolType.Field(i)
			prefixTag := fieldInfo.Tag.Get("prefix")
			fmt.Printf("       %s → %s\n", prefixTag, fieldInfo.Name)
		}
		return
	}

	if len(options["help"]) > 0 || 0 == len(arguments) || 1 != len(options["file"]) {
		exitwithstatus.Message("usage: %s [--help] [--verbose] [--quiet] [--count=N] --file=FILE tag [--list] [key-prefix]", program)
	}

	// stop if prefix no longer matches
	earlyStop := len(options["early"]) > 0

	colour := len(options["colour"]) > 0
	ascii := len(options["ascii"]) > 0
	delete := len(options["delete"]) > 0
	verbose := len(options["verbose"]) > 0

	count := 10
	if len(options["count"]) > 0 {
		count, err = strconv.Atoi(options["count"][0])
		if nil != err {
			exitwithstatus.Message("%s: convert count error: %s", program, err)
		}
		if count < 1 {
			exitwithstatus.Message("%s: invalid count: %d", program, count)
		}
	}

	filename := options["file"][0]
	tag := arguments[0]
	if verbose {
		fmt.Printf("read tag: %s from file: %q\n", tag, filename)
	}

	prefix := []byte(nil)
	if len(arguments) > 1 {
		prefix, err = hex.DecodeString(arguments[1])
		if nil != err {
			exitwithstatus.Message("%s: convert prefix error: %s", program, err)
		}
	}

	logging := logger.Configuration{
		Directory: ".",
		File:      "bitmark-dumpdb.log",
		Size:      1048576,
		Count:     10,
		Console:   true,
		Levels: map[string]string{
			logger.DefaultTag: "critical",
		},
	}

	// start logging
	if err = logger.Initialise(logging); nil != err {
		exitwithstatus.Message("%s: logger setup failed with error: %s", program, err)
	}
	defer logger.Finalise()

	// start of main processing
	_, _, err = storage.Initialise(filename, storage.ReadOnly)
	if nil != err {
		exitwithstatus.Message("%s: storage setup failed with error: %s", program, err)
	}

	defer storage.Finalise()

	// this will be a struct type
	poolType := reflect.TypeOf(storage.Pool)

	// read-only access
	poolValue := reflect.ValueOf(storage.Pool)

	// the handle
	//p := (*storage.PoolHandle)(nil)
	// write access to p as a Value
	//pvalue := reflect.ValueOf(&p).Elem()

	// scan each field to locate tag
	var p reflect.Value
tag_scan:
	for i := 0; i < poolType.NumField(); i += 1 {
		fieldInfo := poolType.Field(i)
		prefixTag := fieldInfo.Tag.Get("prefix")
		if tag == prefixTag {
			//pvalue.Set(poolValue.Field(i))
			p = poolValue.Field(i)
			break tag_scan
		}

	}
	if p.IsNil() {
		exitwithstatus.Message("%s: no pool corresponding to: %q", program, tag)
	}

	// dump the items as hex
	cf := p.MethodByName("NewFetchCursor")
	if !cf.IsValid() {
		exitwithstatus.Message("%s: no cursor access corresponding to: %q", program, tag)
	}

	//cursor := p.NewFetchCursor()

	cursor := (*storage.FetchCursor)(nil)
	// write access to p as a Value
	cValue := reflect.ValueOf(&cursor).Elem()
	cValue.Set(cf.Call(nil)[0])

	if len(prefix) > 0 {
		cursor.Seek(prefix)
	}

	data, err := cursor.Fetch(count)
	if nil != err {
		exitwithstatus.Message("%s: error on Fetch: %s", program, err)
	}

	l := len(prefix)

	ck1 := ""
	ck2 := ""
	cv1 := ""
	cv2 := ""
	cd1 := ""
	cd2 := ""
	cd3 := ""
	cd4 := ""
	cn := ""
	ce := ""
	if colour {
		ck1 = keyColour1
		ck2 = keyColour2
		cv1 = valColour1
		cv2 = valColour2
		cd1 = delColour1
		cd2 = delColour2
		cd3 = delColour3
		cd4 = delColour4
		cn = nodelColour
		ce = endColour
	}
print_loop:
	for i, e := range data {
		if earlyStop && len(e.Key) >= len(prefix) && !bytes.Equal(prefix, e.Key[:l]) {
			fmt.Printf("*** early stop\n")
			break print_loop
		}

		fmt.Printf("%d: %sKey: %s%x%s\n", i, ck1, ck2, e.Key, ce)
		if ascii {
			prefix := fmt.Sprintf("%d: %sVal: %s", i, cv1, cv2)
			suffix := ce
			hexDump(prefix, suffix, e.Value)

		} else {
			fmt.Printf("%d: %sVal: %s%x%s\n", i, cv1, cv2, e.Value, ce)
		}
		if delete {
		delete_loop:
			for {
				fmt.Printf("%d: %sDelete Key: %s%x%s ? [yNq]: ", i, cd1, cd2, e.Key, ce)

				buffer := make([]byte, 100)
				n, err := os.Stdin.Read(buffer)
				if nil != err {
					exitwithstatus.Message("%s: error on Stdin.Read: %e", program, err)
				}

				response := strings.TrimSpace(string(buffer[:n]))
				switch strings.ToLower(response) {

				case "y", "yes":
					//p.Delete(e.Key)
					deleteRecord := p.MethodByName("Delete")
					if !deleteRecord.IsValid() {
						exitwithstatus.Message("%s: no Delete method corresponding to: %q", program, tag)
					}
					deleteRecord.Call([]reflect.Value{reflect.ValueOf(e.Key)})
					fmt.Printf("%d: %s***DELETED: %s%x%s\n", i, cd3, cd4, e.Key, ce)
					break delete_loop

				case "", "n", "no":
					fmt.Printf("%d: %sRetain Key: %s%x%s\n", i, cn, ck2, e.Key, ce)
					break delete_loop

				case "q", "quit", "e", "exit", "x":
					fmt.Printf("Terminated\n")
					return

				default:
					fmt.Printf("Please answer yes or no\n")
				}
			}
		}
	}
}

// dump hex data on stdout
func hexDump(prefix string, suffix string, data []byte) {
	address := 0
	const bytesPerLine = 32
	for i := 0; i < len(data); i += bytesPerLine {
		fmt.Printf("%s%04x  ", prefix, address)
		address += bytesPerLine
		for j := 0; j < bytesPerLine; j += 1 {
			if bytesPerLine/2 == j {
				fmt.Printf(" ")
			}
			if i+j < len(data) {
				fmt.Printf("%02x ", data[i+j])
			} else {
				fmt.Printf("   ")
			}
		}
		fmt.Printf(" |")
	ascii_loop:
		for j := 0; j < bytesPerLine; j += 1 {
			if i+j < len(data) {
				c := data[i+j]
				if c < 32 || c >= 127 {
					c = '.'
				}
				fmt.Printf("%c", c)

			} else {
				break ascii_loop
			}
		}
		fmt.Printf("|%s\n", suffix)
	}
}
