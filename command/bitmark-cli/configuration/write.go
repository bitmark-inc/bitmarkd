// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package configuration

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Save - update configuration file with current data
func Save(filename string, configuration *Configuration) error {

	tempFile := filename + ".new"
	previousFile := filename + ".bk"

	os.Remove(tempFile)

	f, err := os.Create(tempFile)
	if err != nil {
		fmt.Printf("Create file fail: %s\n", err)
		return err
	}

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	err = enc.Encode(configuration)
	if err != nil {
		f.Close()
		return err
	}

	f.Close()

	err = os.Remove(previousFile)
	if err != nil && !strings.Contains(err.Error(), "no such file") {
		return err
	}
	err = os.Rename(filename, previousFile)
	if err != nil && !strings.Contains(err.Error(), "no such file") {
		return err
	}
	err = os.Rename(tempFile, filename)
	if err != nil {
		return err
	}

	return nil
}
