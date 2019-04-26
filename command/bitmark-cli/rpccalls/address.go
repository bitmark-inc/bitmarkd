// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpccalls

import (
	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/keypair"
)

// helper to make an address
func makeAddress(keyPair *keypair.KeyPair, testnet bool) *account.Account {

	return &account.Account{
		AccountInterface: &account.ED25519Account{
			Test:      testnet,
			PublicKey: keyPair.PublicKey[:],
		},
	}
}
