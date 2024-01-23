// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package currency

import (
	"testing"

	"github.com/bitmark-inc/bitmarkd/chain"
)

func TestBtcChainParams(t *testing.T) {

	btcLocalParams := Bitcoin.ChainParam(chain.Local)

	if btcLocalParams.Name != "regtest" {
		t.Fatal("invalid network")
	}

	if btcLocalParams.HDCoinType != 1 {
		t.Fatal("incorrect currency")
	}

	btcTestnetParams := Bitcoin.ChainParam(chain.Testing)

	if btcTestnetParams.Name != "testnet3" {
		t.Fatal("invalid network")
	}

	if btcTestnetParams.HDCoinType != 1 {
		t.Fatal("incorrect currency")
	}

	btcMainnetParams := Bitcoin.ChainParam(chain.Bitmark)

	if btcMainnetParams.Name != "mainnet" {
		t.Fatal("invalid network")
	}

	if btcMainnetParams.HDCoinType != 0 {
		t.Fatal("incorrect currency")
	}
}

func TestLtcChainParams(t *testing.T) {

	ltcLocalParams := Litecoin.ChainParam(chain.Local)

	if ltcLocalParams.Name != "regtest" {
		t.Fatal("invalid network")
	}

	if ltcLocalParams.HDCoinType != 1 {
		t.Fatal("incorrect currency")
	}

	ltcTestnetParams := Litecoin.ChainParam(chain.Testing)

	if ltcTestnetParams.Name != "testnet4" {
		t.Fatal("invalid network")
	}

	if ltcTestnetParams.HDCoinType != 1 {
		t.Fatal("incorrect currency")
	}

	ltcMainnetParams := Litecoin.ChainParam(chain.Bitmark)

	if ltcMainnetParams.Name != "mainnet" {
		t.Fatal("invalid network")
	}

	if ltcMainnetParams.HDCoinType != 2 {
		t.Fatal("incorrect currency")
	}
}
