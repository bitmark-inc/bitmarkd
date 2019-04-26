// Copyright (c) 2014-2019 Bitmark Inc.
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
	if nil != err {
		fmt.Printf("Create file fail: %s\n", err)
		return err
	}

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	err = enc.Encode(configuration)
	if nil != err {
		f.Close()
		return err
	}

	f.Close()

	err = os.Remove(previousFile)
	if nil != err && !strings.Contains(err.Error(), "no such file") {
		return err
	}
	err = os.Rename(filename, previousFile)
	if nil != err && !strings.Contains(err.Error(), "no such file") {
		return err
	}
	err = os.Rename(tempFile, filename)
	if nil != err {
		return err
	}

	return nil
}
