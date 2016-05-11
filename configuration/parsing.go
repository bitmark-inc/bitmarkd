// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package configuration

import (
	"path/filepath"
)

const (
	DefaultNetwork = "testing"
)

type PrivateKeyConfig struct {
	Iter int    `libucl:"iter"`
	Salt string `libucl:"salt"`
}

type IdentityType struct {
	Name               string           `libucl:"name"`
	Description        string           `libucl:"description"`
	Public_key         string           `libucl:"public_key"`
	Private_key        string           `libucl:"private_key"`
	Private_key_config PrivateKeyConfig `libucl:"private_key_config"`
}

type Configuration struct {
	Default_identity string         `libucl:"default_identity"`
	Network          string         `libucl:"network"`
	Connect          string         `libucl:"connect"`
	Identities       []IdentityType `libucl:"identities"`
}

func GetConfiguration(configurationFileName string) (*Configuration, error) {

	configurationFileName, err := filepath.Abs(filepath.Clean(configurationFileName))
	if nil != err {
		return nil, err
	}

	options := &Configuration{}
	if err := readConfigurationFile(configurationFileName, options); err != nil {
		return nil, err
	}

	return options, nil
}
