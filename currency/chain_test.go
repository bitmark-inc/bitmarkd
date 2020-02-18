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

	if "regtest" != btcLocalParams.Name {
		t.Fatal("invalid network")
	}

	if 1 != btcLocalParams.HDCoinType {
		t.Fatal("incorrect currency")
	}

	btcTestnetParams := Bitcoin.ChainParam(chain.Testing)

	if "testnet3" != btcTestnetParams.Name {
		t.Fatal("invalid network")
	}

	if 1 != btcTestnetParams.HDCoinType {
		t.Fatal("incorrect currency")
	}

	btcMainnetParams := Bitcoin.ChainParam(chain.Bitmark)

	if "mainnet" != btcMainnetParams.Name {
		t.Fatal("invalid network")
	}

	if 0 != btcMainnetParams.HDCoinType {
		t.Fatal("incorrect currency")
	}
}

func TestLtcChainParams(t *testing.T) {

	ltcLocalParams := Litecoin.ChainParam(chain.Local)

	if "regtest" != ltcLocalParams.Name {
		t.Fatal("invalid network")
	}

	if 1 != ltcLocalParams.HDCoinType {
		t.Fatal("incorrect currency")
	}

	ltcTestnetParams := Litecoin.ChainParam(chain.Testing)

	if "testnet4" != ltcTestnetParams.Name {
		t.Fatal("invalid network")
	}

	if 1 != ltcTestnetParams.HDCoinType {
		t.Fatal("incorrect currency")
	}

	ltcMainnetParams := Litecoin.ChainParam(chain.Bitmark)

	if "mainnet" != ltcMainnetParams.Name {
		t.Fatal("invalid network")
	}

	if 2 != ltcMainnetParams.HDCoinType {
		t.Fatal("incorrect currency")
	}
}
