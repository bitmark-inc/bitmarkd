// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpccalls

import (
	"encoding/json"
	"fmt"
)

func (client *Client) printJson(title string, message interface{}) error {

	if !client.verbose {
		return nil
	}

	b, err := json.MarshalIndent(message, "", "  ")
	if nil != err {
		return err
	}

	if "" == title {
		fmt.Fprintf(client.handle, "%s\n", b)
	} else {
		fmt.Fprintf(client.handle, "%s:\n%s\n", title, b)
	}
	return nil
}
