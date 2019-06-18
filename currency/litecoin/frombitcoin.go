// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package litecoin

import (
	"crypto/sha256"

	"github.com/bitmark-inc/bitmarkd/currency/bitcoin"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
)

// FromBitcoin - check the address and return its version
func FromBitcoin(address string) (string, error) {
	version, addressBytes, err := bitcoin.ValidateAddress(address)

	if nil != err {
		return "", err
	}

	var ltcVersion Version
	switch version {
	case bitcoin.Livenet:
		ltcVersion = Livenet
	case bitcoin.LivenetScript:
		ltcVersion = LivenetScript
	// case bitcoin.LivenetScript2:
	// 	ltcVersion = LivenetScript2
	case bitcoin.Testnet:
		ltcVersion = Testnet
	case bitcoin.TestnetScript:
		ltcVersion = TestnetScript
	default:
		return "", fault.ErrInvalidBitcoinAddress
	}

	ltc := append([]byte{byte(ltcVersion)}, addressBytes[:]...)

	h := sha256.New()
	h.Write(ltc)
	d := h.Sum([]byte{})
	h = sha256.New()
	h.Write(d)
	d = h.Sum([]byte{})

	ltc = append(ltc, d[0:4]...)

	return util.ToBase58(ltc), nil
}
