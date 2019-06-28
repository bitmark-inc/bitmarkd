// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package configuration

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Configuration - configuration file data format
type Configuration struct {
	DefaultIdentity string              `json:"default_identity"`
	TestNet         bool                `json:"testnet"`
	Connections     []string            `json:"connections"`
	Identities      map[string]Identity `json:"identities"`
}

// Identity - mix of plain and encrypted data
type Identity struct {
	Description string `json:"description"`
	Account     string `json:"account"`
	Data        string `json:"data"`
	Salt        string `json:"salt"`
}

// Read - read the configuration
func Read(filename string) (*Configuration, error) {

	options := &Configuration{}

	err := readConfiguration(filename, options)
	if nil != err {
		return nil, err
	}
	return options, nil
}

// generic JSON decoder
func readConfiguration(filename string, options interface{}) error {

	filename, err := filepath.Abs(filepath.Clean(filename))
	if nil != err {
		return err
	}

	f, err := os.Open(filename)
	if nil != err {
		return err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	err = dec.Decode(options)
	if nil != err {
		return err
	}

	return nil
}
