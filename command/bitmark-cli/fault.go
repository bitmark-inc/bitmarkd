// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"github.com/bitmark-inc/bitmarkd/fault"
)

// common errors - keep in alphabetic order
const (
	ErrKeyLength        = fault.InvalidError("key length is invalid")
	ErrNotFoundIdentity = fault.NotFoundError("identity name not found")
	ErrInvalidNetwork   = fault.InvalidError("invalid network")
	ErrNilKeyPair       = fault.ProcessError("internal error: nil key pair")
)
