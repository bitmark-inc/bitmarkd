// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package configuration

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/fault"
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

// Load - read the configuration
func Load(filename string) (*Configuration, error) {

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

// Identity - find identity for a given name
func (config *Configuration) Identity(name string) (*Identity, error) {
	id, ok := config.Identities[name]
	if !ok {
		return nil, fault.ErrIdentityNameNotFound
	}

	return &id, nil
}

// Account - find identity for a given name and convert to an account
func (config *Configuration) Account(name string) (*account.Account, error) {
	id, err := config.Identity(name)
	if nil != err {
		return nil, err
	}

	acc, err := account.AccountFromBase58(id.Account)

	return acc, err
}

// Private - find identity decrypt all data for a given name
func (config *Configuration) Private(password string, name string) (*Private, error) {
	id, err := config.Identity(name)
	if nil != err {
		return nil, err
	}

	return decryptIdentity(password, id)
}

// AddIdentity - store encrypted identity
func (config *Configuration) AddIdentity(name string, description string, seed string, password string) error {

	if _, ok := config.Identities[name]; ok {
		return fault.ErrIdentityNameAlreadyExists
	}

	salt, secretKey, err := hashPassword(password)
	if nil != err {
		return err
	}

	encrypted, err := encryptData(seed, secretKey)
	if nil != err {
		return err
	}

	private, err := account.PrivateKeyFromBase58Seed(seed)
	if nil != err {
		return err
	}

	config.Identities[name] = Identity{
		Description: description,
		Account:     private.Account().String(),
		Data:        encrypted,
		Salt:        salt.String(),
	}

	return nil
}

// AddReceiveOnlyIdentity - store public-only identity
func (config *Configuration) AddReceiveOnlyIdentity(name string, description string, acc string) error {

	if _, ok := config.Identities[name]; ok {
		return fault.ErrIdentityNameAlreadyExists
	}

	_, err := account.AccountFromBase58(acc)
	if nil != err {
		return err
	}

	config.Identities[name] = Identity{
		Description: description,
		Account:     acc,
		Data:        "",
		Salt:        "",
	}

	return nil
}
