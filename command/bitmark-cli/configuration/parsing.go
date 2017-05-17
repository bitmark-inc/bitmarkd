// Copyright (c) 2014-2017 Bitmark Inc.
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
	Salt string `libucl:"salt"`
}

// full access to data (includes private data)

type IdentityType struct {
	Name               string           `libucl:"name"`
	Description        string           `libucl:"description"`
	Public_key         string           `libucl:"public_key"`
	Private_key        string           `libucl:"private_key"`
	Seed               string           `libucl:"seed"`
	Private_key_config PrivateKeyConfig `libucl:"private_key_config"`
}

type Configuration struct {
	Default_identity string         `libucl:"default_identity"`
	Network          string         `libucl:"network"`
	Connect          string         `libucl:"connect"`
	Identity         []IdentityType `libucl:"identity"`
}

// form of configuration in the config file
// used by write.go
const configurationTemplate = `# bitmark-cli.conf -*- mode: libucl -*-

default_identity = "{{.Default_identity}}"

network = "{{.Network}}"
connect = "{{.Connect}}"

{{range .Identity}}
identity {
  name = "{{.Name}}"
  description = "{{.Description}}"
  public_key = "{{.Public_key}}"
  private_key = "{{.Private_key}}"
  seed = "{{.Seed}}"
  private_key_config {
    salt = "{{.Private_key_config.Salt}}"
  }
}
{{end}}
`

// restricted access to data (excludes private items)

type InfoIdentityType struct {
	Name        string `libucl:"name"`
	Description string `libucl:"description"`
	Public_key  string `libucl:"public_key"`
	Account     string
}

type InfoConfiguration struct {
	Default_identity string             `libucl:"default_identity"`
	Network          string             `libucl:"network"`
	Connect          string             `libucl:"connect"`
	Identity         []InfoIdentityType `libucl:"identity"`
}

// full access to data (includes private data)
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

// restricted access to data (excludes private items)
func GetInfoConfiguration(configurationFileName string) (*InfoConfiguration, error) {
	configurationFileName, err := filepath.Abs(filepath.Clean(configurationFileName))
	if nil != err {
		return nil, err
	}

	options := &InfoConfiguration{}
	if err := readConfigurationFile(configurationFileName, options); err != nil {
		return nil, err
	}

	return options, nil
}
