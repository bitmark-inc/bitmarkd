// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"

	"golang.org/x/crypto/sha3"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Printf("usage: makeproof payId payNonce\n")
		return
	}

	payId := toHex(os.Args[1])
	payNonce := toHex(os.Args[2])

	nonce := uint64(12345)
	nonceBuffer := make([]byte, 8)

	for {
		nonce += 113113
		binary.BigEndian.PutUint64(nonceBuffer, nonce)

		// compute hash
		h := sha3.New256()
		h.Write(payId)
		h.Write(payNonce)
		h.Write(nonceBuffer)
		var digest [32]byte
		h.Sum(digest[:0])
		if digest[0]|digest[1]|digest[2] == 0 {
			fmt.Printf("possible nonce: %x  with digest: %x\n", nonceBuffer, digest)
		}
	}
}

func toHex(s string) []byte {

	size := hex.DecodedLen(len(s))

	buffer := make([]byte, size)
	byteCount, err := hex.Decode(buffer, []byte(s))
	if err != nil {
		fmt.Printf("hex decode error: %s\n", err)
		panic("hex error")
	}
	if byteCount != size {
		panic("hex size mismatch")
	}

	return buffer
}
