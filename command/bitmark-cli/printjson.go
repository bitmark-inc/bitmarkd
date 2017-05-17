// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"github.com/bitmark-inc/exitwithstatus"
	"os"
)

func printJson(title string, message interface{}, print ...bool) {

	// check otional verbose flag
	if 0 != len(print) {
		if !print[0] {
			return
		}
	}
	b, err := json.MarshalIndent(message, "", "  ")
	if nil != err {
		exitwithstatus.Message("Error: printjson marshall error: %s", err)
	}

	if "" == title {
		fmt.Printf("%s\n", b)
	} else {
		fmt.Printf("%s:\n%s\n", title, b)
	}
}

// output a JSON block to a file
func printJsonToFile(filename string, message interface{}) {

	b, err := json.MarshalIndent(message, "", "  ")
	if nil != err {
		exitwithstatus.Message("Error: printjson marshall error: %s", err)
	}
	file, err := os.Create(filename)
	if nil != err {
		exitwithstatus.Message("Error: printjson file create error: %s", err)
	}
	_, err = file.Write(b)
	file.Close()
	if nil != err {
		exitwithstatus.Message("Error: printjson file write error: %s", err)
	}
}
