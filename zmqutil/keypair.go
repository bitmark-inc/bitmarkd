// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package zmqutil

import (
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
	zmq "github.com/pebbe/zmq4"
	"io/ioutil"
	"os"
)

// create a new public/private keypair and write them to separate
// files
//
// keyare encoded in in Z85 (ZeroMQ Base-85 Encoding) see: http://rfc.zeromq.org/spec:32
func MakeKeyPair(publicKeyFileName string, privateKeyFileName string) error {
	if util.EnsureFileExists(publicKeyFileName) {
		return fault.ErrKeyFileAlreadyExists
	}

	if util.EnsureFileExists(privateKeyFileName) {
		return fault.ErrKeyFileAlreadyExists
	}

	publicKey, privateKey, err := zmq.NewCurveKeypair()
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

// read a key from a file returning it as a Z85 string
func ReadKeyFile(keyFileName string) (string, error) {
	if !util.EnsureFileExists(keyFileName) {
		return "", fault.ErrKeyFileNotFound
	}
	data, err := ioutil.ReadFile(keyFileName)
	if err != nil {
		return "", err
	}

	return string(data), nil
}
