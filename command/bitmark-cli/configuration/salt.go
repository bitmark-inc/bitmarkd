// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package configuration

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/bitmark-inc/bitmarkd/fault"
)

const (
	saltSize = 32
)

// Salt - type to hold a salt value
type Salt [saltSize]byte

// MakeSalt - create a salt using secure random number generator
func MakeSalt() (*Salt, error) {
	salt := new(Salt)
	if _, err := rand.Read(salt[:]); err != nil {
		return nil, err
	}
	return salt, nil
}

// Bytes - convert a binary salt to byte slice
func (salt Salt) Bytes() []byte {
	return salt[:]
}

// String - convert a binary salt to little endian hex string for use by the fmt package (for %s)
func (salt Salt) String() string {
	return hex.EncodeToString(salt.Bytes())
}

// MarshalText - convert salt to little endian hex text
//
// ***** possibly use NewEncoder and byte buffer to save copy
func (salt *Salt) MarshalText() []byte {
	// encode to hex
	size := hex.EncodedLen(saltSize)
	buffer := make([]byte, size)
	hex.Encode(buffer, salt.Bytes())

	return buffer
}

// UnmarshalText - convert little endian hex text into a salt
func (salt *Salt) UnmarshalText(s []byte) error {
	buffer := make([]byte, hex.DecodedLen(len(s)))
	byteCount, err := hex.Decode(buffer, s)
	if err != nil {
		return err
	}

	if saltSize != byteCount {
		return fault.UnmarshalTextFailed
	}
	copy(salt[:], buffer)
	return nil
}
