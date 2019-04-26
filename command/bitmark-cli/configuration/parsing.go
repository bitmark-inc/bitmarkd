// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package configuration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/encrypt"
)

// DefaultNetwork - select the default network
const DefaultNetwork = "testing"

// Configuration - configuration file data format
type Configuration struct {
	DefaultIdentity string                 `json:"default_identity"`
	TestNet         bool                   `json:"testnet"`
	Connect         string                 `json:"connect"`
	Identities      []encrypt.IdentityType `json:"identities"`
}

// InfoIdentityType - restricted access to data (excludes private items)
type InfoIdentityType struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	PublicKey   string `json:"public_key"`
	Account     string `json:"account"`
}

// InfoConfiguration - restricted view of configuration
type InfoConfiguration struct {
	DefaultIdentity string             `json:"default_identity"`
	TestNet         bool               `json:"testnet"`
	Connect         string             `json:"connect"`
	Identities      []InfoIdentityType `json:"identities"`
}

func (s *InfoConfiguration) Len() int {
	return len(s.Identities)
}

func (s *InfoConfiguration) Swap(i, j int) {
	s.Identities[i], s.Identities[j] = s.Identities[j], s.Identities[i]
}

func (s *InfoConfiguration) Less(i int, j int) bool {
	return s.Identities[i].Name < s.Identities[j].Name
}

// GetConfiguration - full access to data (includes private data)
func GetConfiguration(filename string) (*Configuration, error) {

	options := &Configuration{}

	err := readConfiguration(filename, options)
	if nil != err {
		return nil, err
	}
	return options, nil
}

// GetInfoConfiguration - restricted access to data (excludes private items)
func GetInfoConfiguration(filename string) (*InfoConfiguration, error) {

	options := &InfoConfiguration{}

	err := readConfiguration(filename, options)
	if nil != err {
		return nil, err
	}

	sort.Sort(options)

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
