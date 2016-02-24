// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"github.com/bitmark-inc/bilateralrpc"
	"github.com/bitmark-inc/bitmarkd/fault"
	"io/ioutil"
	"os"
)

// create a new public/private keypair
func makeKeyPair(name string, publicKeyFileName string, privateKeyFileName string) error {
	if ensureFileExists(publicKeyFileName) {
		return fault.ErrKeyFileAlreadyExists
	}

	if ensureFileExists(privateKeyFileName) {
		return fault.ErrKeyFileAlreadyExists
	}

	publicKey, privateKey, err := bilateralrpc.NewKeypair()
	if nil != err {
		return err
	}

	if err = ioutil.WriteFile(publicKeyFileName, []byte(publicKey), 0666); err != nil {
		return err
	}

	if err = ioutil.WriteFile(privateKeyFileName, []byte(privateKey), 0600); err != nil {
		os.Remove(publicKeyFileName)
		return err
	}

	return nil
}

// read a key from a file
func readKeyFile(keyFileName string) (string, error) {
	if !ensureFileExists(keyFileName) {
		return "", fault.ErrKeyFileNotFound
	}
	data, err := ioutil.ReadFile(keyFileName)
	if err != nil {
		return "", err
	}

	return string(data), nil
}
