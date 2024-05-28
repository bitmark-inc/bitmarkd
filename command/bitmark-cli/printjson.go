// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"io"
)

func printJson(handle io.Writer, message interface{}) error {

	prefix := ""
	indent := "  "

	b, err := json.MarshalIndent(message, prefix, indent)
	if err != nil {
		return err
	}

	fmt.Fprintf(handle, "%s\n", b)
	return nil
}
