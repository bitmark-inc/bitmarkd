// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"github.com/bitmark-inc/exitwithstatus"
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
		exitwithstatus.Message("Error: printJson failed: %s", err)

	}

	if "" == title {
		fmt.Printf("%s\n", b)
	} else {
		fmt.Printf("%s:\n%s\n", title, b)
	}
}
