// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package configuration

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/bitmark-inc/bitmark-cli/fault"
	"io"
)

const (
	saltSize = 16
)

type Salt [saltSize]byte

func MakeSalt() (*Salt, error) {
	salt := new(Salt)
	if _, err := io.ReadFull(rand.Reader, salt[:]); err != nil {
		return salt, err
	}
	return salt, nil
}

// convert a binary salt to byte slice
func (salt Salt) Bytes() []byte {
	return salt[:]
}

// convert a binary salt to little endian hex string for use by the fmt package (for %s)
func (salt Salt) String() string {
	return hex.EncodeToString(salt.Bytes())
}

// convert salt to little endian hex text
//
// ***** possibly use NewEncoder and byte buffer to save copy
func (salt *Salt) MarshalText() []byte {
	// encode to hex
	size := hex.EncodedLen(saltSize)
	buffer := make([]byte, size)
	hex.Encode(buffer, salt.Bytes())

	return buffer
}

// convert little endian hex text into a salt
func (salt *Salt) UnmarshalText(s []byte) error {
	buffer := make([]byte, hex.DecodedLen(len(s)))
	byteCount, err := hex.Decode(buffer, s)
	if nil != err {
		return err
	}

	if saltSize != byteCount {
		fmt.Printf("invalid byte\n")
		return fault.ErrUnmarshalTextFail
	}
	copy(salt[:], buffer)
	return nil
}
