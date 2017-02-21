// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package zmqutil

import (
	"encoding/hex"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
	zmq "github.com/pebbe/zmq4"
	"io/ioutil"
	"os"
	"strings"
)

const (
	taggedPublic  = "PUBLIC:"
	taggedPrivate = "PRIVATE:"
	publicLength  = 32
	privateLength = 32
)

// create a new public/private keypair and write them to separate
// files
func MakeKeyPair(publicKeyFileName string, privateKeyFileName string) error {
	if util.EnsureFileExists(publicKeyFileName) {
		return fault.ErrKeyFileAlreadyExists
	}

	if util.EnsureFileExists(privateKeyFileName) {
		return fault.ErrKeyFileAlreadyExists
	}

	// keys are encoded in in Z85 (ZeroMQ Base-85 Encoding) see: http://rfc.zeromq.org/spec:32
	publicKey, privateKey, err := zmq.NewCurveKeypair()
	if nil != err {
		return err
	}

	publicKey = taggedPublic + hex.EncodeToString([]byte(zmq.Z85decode(publicKey))) + "\n"
	privateKey = taggedPrivate + hex.EncodeToString([]byte(zmq.Z85decode(privateKey))) + "\n"

	if err = ioutil.WriteFile(publicKeyFileName, []byte(publicKey), 0666); err != nil {
		return err
	}

	if err = ioutil.WriteFile(privateKeyFileName, []byte(privateKey), 0600); err != nil {
		os.Remove(publicKeyFileName)
		return err
	}

	return nil
}

// read a public key from a file returning it as a 32 byte string
func ReadPublicKeyFile(keyFileName string) ([]byte, error) {
	data, private, err := ReadKeyFile(keyFileName)
	if err != nil {
		return []byte{}, err
	}
	if private {
		return []byte{}, fault.ErrInvalidPublicKeyFile
	}
	return data, err
}

// read a private key from a file returning it as a 32 byte string
func ReadPrivateKeyFile(keyFileName string) ([]byte, error) {
	data, private, err := ReadKeyFile(keyFileName)
	if err != nil {
		return []byte{}, err
	}
	if !private {
		return []byte{}, fault.ErrInvalidPrivateKeyFile
	}
	return data, err
}

// read a public key from a file returning it as a 32 byte string
func ReadKeyFile(keyFileName string) ([]byte, bool, error) {
	if !util.EnsureFileExists(keyFileName) {
		return []byte{}, false, fault.ErrKeyFileNotFound
	}
	data, err := ioutil.ReadFile(keyFileName)
	if err != nil {
		return []byte{}, false, err
	}

	s := strings.TrimSpace(string(data))
	if strings.HasPrefix(s, taggedPrivate) {
		h, err := hex.DecodeString(s[len(taggedPrivate):])
		if err != nil {
			return []byte{}, false, err
		}
		if len(h) != privateLength {
			return []byte{}, false, fault.ErrInvalidPrivateKeyFile
		}
		return h, true, nil
	} else if strings.HasPrefix(s, taggedPublic) {
		h, err := hex.DecodeString(s[len(taggedPublic):])
		if err != nil {
			return []byte{}, false, err
		}
		if len(h) != publicLength {
			return []byte{}, false, fault.ErrInvalidPublicKeyFile
		}
		return h, false, nil
	}

	return []byte{}, false, fault.ErrInvalidPublicKeyFile
}
