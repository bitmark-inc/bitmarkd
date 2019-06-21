// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package chain

// names of all chains
const (
	Bitmark = "bitmark"
	Testing = "testing"
	Local   = "local"
)

// Valid - validate a chain name
func Valid(name string) bool {
	switch name {
	case Bitmark, Testing, Local:
		return true
	default:
		return false
	}
}
