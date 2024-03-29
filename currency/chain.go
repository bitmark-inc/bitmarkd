// SPDX-License-Identifier: ISC
// Copyright (c) 2013-2017 The btcsuite developers
// Copyright (c) 2015-2016 The Decred developers
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package currency

import (
	"math"
	"math/big"
	"time"

	"github.com/bitmark-inc/bitmarkd/chain"
	"github.com/bitmark-inc/logger"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
)

var btcMainNetParams *chaincfg.Params = &chaincfg.MainNetParams
var btcTestNet3Params *chaincfg.Params = &chaincfg.TestNet3Params
var btcRegressionNetParams *chaincfg.Params = &chaincfg.RegressionNetParams

// A group of variables for Litecoin Mainnet
var (
	genesisCoinbaseTx = wire.MsgTx{
		Version: 1,
		TxIn: []*wire.TxIn{
			{
				PreviousOutPoint: wire.OutPoint{
					Hash:  chainhash.Hash{},
					Index: 0xffffffff,
				},
				SignatureScript: []byte{
					0x04, 0xff, 0xff, 0x00, 0x1d, 0x01, 0x04, 0x40, 0x4e, 0x59, 0x20, 0x54, 0x69, 0x6d, 0x65, 0x73, // |.......@NY Times|
					0x20, 0x30, 0x35, 0x2f, 0x4f, 0x63, 0x74, 0x2f, 0x32, 0x30, 0x31, 0x31, 0x20, 0x53, 0x74, 0x65, // | 05/Oct/2011 Ste|
					0x76, 0x65, 0x20, 0x4a, 0x6f, 0x62, 0x73, 0x2c, 0x20, 0x41, 0x70, 0x70, 0x6c, 0x65, 0xe2, 0x80, // |ve Jobs, Apple..|
					0x99, 0x73, 0x20, 0x56, 0x69, 0x73, 0x69, 0x6f, 0x6e, 0x61, 0x72, 0x79, 0x2c, 0x20, 0x44, 0x69, // |.s Visionary, Di|
					0x65, 0x73, 0x20, 0x61, 0x74, 0x20, 0x35, 0x36, // |es at 56|

				},
				Sequence: 0xffffffff,
			},
		},
		TxOut: []*wire.TxOut{
			{
				Value: 0x12a05f200,
				PkScript: []byte{
					0x41, 0x4, 0x1, 0x84, 0x71, 0xf, 0xa6, 0x89,
					0xad, 0x50, 0x23, 0x69, 0xc, 0x80, 0xf3, 0xa4,
					0x9c, 0x8f, 0x13, 0xf8, 0xd4, 0x5b, 0x8c, 0x85,
					0x7f, 0xbc, 0xbc, 0x8b, 0xc4, 0xa8, 0xe4, 0xd3,
					0xeb, 0x4b, 0x10, 0xf4, 0xd4, 0x60, 0x4f, 0xa0,
					0x8d, 0xce, 0x60, 0x1a, 0xaf, 0xf, 0x47, 0x2,
					0x16, 0xfe, 0x1b, 0x51, 0x85, 0xb, 0x4a, 0xcf,
					0x21, 0xb1, 0x79, 0xc4, 0x50, 0x70, 0xac, 0x7b,
					0x3, 0xa9, 0xac,
				},
			},
		},
		LockTime: 0,
	}

	mainPowLimit, _ = new(big.Int).SetString("0x0fffff000000000000000000000000000000000000000000000000000000", 0)
	// genesisHash is the hash of the first block in the block chain for the main
	// network (genesis block).
	genesisHash = chainhash.Hash([chainhash.HashSize]byte{ // Make go vet happy.
		0xe2, 0xbf, 0x04, 0x7e, 0x7e, 0x5a, 0x19, 0x1a,
		0xa4, 0xef, 0x34, 0xd3, 0x14, 0x97, 0x9d, 0xc9,
		0x98, 0x6e, 0x0f, 0x19, 0x25, 0x1e, 0xda, 0xba,
		0x59, 0x40, 0xfd, 0x1f, 0xe3, 0x65, 0xa7, 0x12,
	})

	// genesisMerkleRoot is the hash of the first transaction in the genesis block
	// for the main network.
	genesisMerkleRoot = chainhash.Hash([chainhash.HashSize]byte{ // Make go vet happy.
		0xd9, 0xce, 0xd4, 0xed, 0x11, 0x30, 0xf7, 0xb7,
		0xfa, 0xad, 0x9b, 0xe2, 0x53, 0x23, 0xff, 0xaf,
		0xa3, 0x32, 0x32, 0xa1, 0x7c, 0x3e, 0xdf, 0x6c,
		0xfd, 0x97, 0xbe, 0xe6, 0xba, 0xfb, 0xdd, 0x97,
	})

	// genesisBlock defines the genesis block of the block chain which serves as the
	// public transaction ledger for the main network.
	genesisBlock = wire.MsgBlock{
		Header: wire.BlockHeader{
			Version:    1,
			PrevBlock:  chainhash.Hash{},  // 0000000000000000000000000000000000000000000000000000000000000000
			MerkleRoot: genesisMerkleRoot, // 97ddfbbae6be97fd6cdf3e7ca13232a3afff2353e29badfab7f73011edd4ced9
			Timestamp:  time.Unix(1317972665, 0),
			Bits:       0x1e0ffff0,
			Nonce:      2084524493,
		},
		Transactions: []*wire.MsgTx{&genesisCoinbaseTx},
	}

	ltcMainNetParams = &chaincfg.Params{
		Name:        "mainnet",
		Net:         wire.BitcoinNet(0xdbb6c0fb),
		DefaultPort: "9333",
		DNSSeeds: []chaincfg.DNSSeed{
			{Host: "seed-a.litecoin.loshan.co.uk", HasFiltering: true},
			{Host: "dnsseed.thrasher.io", HasFiltering: true},
			{Host: "dnsseed.litecointools.com", HasFiltering: false},
			{Host: "dnsseed.litecoinpool.org", HasFiltering: false},
			{Host: "dnsseed.koin-project.com", HasFiltering: false},
		},

		// Chain parameters
		GenesisBlock:             &genesisBlock,
		GenesisHash:              &genesisHash,
		PowLimit:                 mainPowLimit,
		PowLimitBits:             504365055,
		BIP0034Height:            710000,
		BIP0065Height:            918684,
		BIP0066Height:            811879,
		CoinbaseMaturity:         100,
		SubsidyReductionInterval: 840000,
		TargetTimespan:           time.Hour*24*3 + time.Hour*12,
		TargetTimePerBlock:       time.Minute*2 + time.Second*30,
		RetargetAdjustmentFactor: 4, // 25% less, 400% more
		ReduceMinDifficulty:      false,
		MinDiffReductionTime:     0,
		GenerateSupported:        false,

		// Checkpoints ordered from oldest to newest.
		Checkpoints: []chaincfg.Checkpoint{},

		// Consensus rule change deployments.
		//
		// The miner confirmation window is defined as:
		//   target proof of work timespan / target proof of work spacing
		RuleChangeActivationThreshold: 6048, // 75% of MinerConfirmationWindow
		MinerConfirmationWindow:       8064, //
		Deployments: [chaincfg.DefinedDeployments]chaincfg.ConsensusDeployment{
			chaincfg.DeploymentTestDummy: {
				BitNumber: 28,
				DeploymentStarter: chaincfg.NewMedianTimeDeploymentStarter(
					time.Unix(1199145601, 0), // January 1, 2008 UTC
				),
				DeploymentEnder: chaincfg.NewMedianTimeDeploymentEnder(
					time.Unix(1230767999, 0), // December 31, 2008 UTC
				),
			},
			chaincfg.DeploymentCSV: {
				BitNumber: 0,
				DeploymentStarter: chaincfg.NewMedianTimeDeploymentStarter(
					time.Unix(1485561600, 0), // January 28, 2017 UTC
				),
				DeploymentEnder: chaincfg.NewMedianTimeDeploymentEnder(
					time.Unix(1517356801, 0), // January 31st, 2018 UTC
				),
			},
			chaincfg.DeploymentSegwit: {
				BitNumber: 1,
				DeploymentStarter: chaincfg.NewMedianTimeDeploymentStarter(
					time.Unix(1485561600, 0), // January 28, 2017 UTC
				),
				DeploymentEnder: chaincfg.NewMedianTimeDeploymentEnder(
					time.Unix(1517356801, 0), // January 31st, 2018 UTC.
				),
			},
		},

		// Mempool parameters
		RelayNonStdTxs: false,

		// Human-readable part for Bech32 encoded segwit addresses, as defined in
		// BIP 173.
		Bech32HRPSegwit: "ltc", // always ltc for main net

		// Address encoding magics
		PubKeyHashAddrID:        0x30, // starts with L
		ScriptHashAddrID:        0x32, // starts with M
		PrivateKeyID:            0xB0, // starts with 6 (uncompressed) or T (compressed)
		WitnessPubKeyHashAddrID: 0x06, // starts with p2
		WitnessScriptHashAddrID: 0x0A, // starts with 7Xh

		// BIP32 hierarchical deterministic extended key magics
		HDPrivateKeyID: [4]byte{0x04, 0x88, 0xad, 0xe4}, // starts with xprv
		HDPublicKeyID:  [4]byte{0x04, 0x88, 0xb2, 0x1e}, // starts with xpub

		// BIP44 coin type used in the hierarchical deterministic path for
		// address generation.
		HDCoinType: 2,
	}
)

// A group of variables for Litecoin Testnet
var (
	testNet4PowLimit, _ = new(big.Int).SetString("0x0fffff000000000000000000000000000000000000000000000000000000", 0)
	testNet4GenesisHash = chainhash.Hash([chainhash.HashSize]byte{ // Make go vet happy.
		0xa0, 0x29, 0x3e, 0x4e, 0xeb, 0x3d, 0xa6, 0xe6,
		0xf5, 0x6f, 0x81, 0xed, 0x59, 0x5f, 0x57, 0x88,
		0xd, 0x1a, 0x21, 0x56, 0x9e, 0x13, 0xee, 0xfd,
		0xd9, 0x51, 0x28, 0x4b, 0x5a, 0x62, 0x66, 0x49,
	})
	testNet4GenesisMerkleRoot = chainhash.Hash([chainhash.HashSize]byte{ // Make go vet happy.
		0xd9, 0xce, 0xd4, 0xed, 0x11, 0x30, 0xf7, 0xb7,
		0xfa, 0xad, 0x9b, 0xe2, 0x53, 0x23, 0xff, 0xaf,
		0xa3, 0x32, 0x32, 0xa1, 0x7c, 0x3e, 0xdf, 0x6c,
		0xfd, 0x97, 0xbe, 0xe6, 0xba, 0xfb, 0xdd, 0x97,
	})

	// testNet4GenesisBlock defines the genesis block of the block chain which
	// serves as the public transaction ledger for the test network (version 4).
	testNet4GenesisBlock = wire.MsgBlock{
		Header: wire.BlockHeader{
			Version:    1,
			PrevBlock:  chainhash.Hash{},          // 0000000000000000000000000000000000000000000000000000000000000000
			MerkleRoot: testNet4GenesisMerkleRoot, // 97ddfbbae6be97fd6cdf3e7ca13232a3afff2353e29badfab7f73011edd4ced9
			Timestamp:  time.Unix(1486949366, 0),
			Bits:       0x1e0ffff0,
			Nonce:      293345,
		},
		Transactions: []*wire.MsgTx{&genesisCoinbaseTx},
	}

	ltcTestNet4Params = &chaincfg.Params{
		Name:        "testnet4",
		Net:         wire.BitcoinNet(0xf1c8d2fd),
		DefaultPort: "19335",
		DNSSeeds: []chaincfg.DNSSeed{
			{Host: "testnet-seed.litecointools.com", HasFiltering: false},
			{Host: "seed-b.litecoin.loshan.co.uk", HasFiltering: true},
			{Host: "dnsseed-testnet.thrasher.io", HasFiltering: true},
		},

		// Chain parameters
		GenesisBlock:             &testNet4GenesisBlock,
		GenesisHash:              &testNet4GenesisHash,
		PowLimit:                 testNet4PowLimit,
		PowLimitBits:             504365055,
		BIP0034Height:            76,
		BIP0065Height:            76,
		BIP0066Height:            76,
		CoinbaseMaturity:         100,
		SubsidyReductionInterval: 840000,
		TargetTimespan:           time.Hour*24*3 + time.Hour*12,
		TargetTimePerBlock:       time.Minute*2 + time.Second*30,
		RetargetAdjustmentFactor: 4, // 25% less, 400% more
		ReduceMinDifficulty:      true,
		MinDiffReductionTime:     time.Minute * 5, // TargetTimePerBlock * 2
		GenerateSupported:        false,

		// Checkpoints ordered from oldest to newest.
		Checkpoints: []chaincfg.Checkpoint{},

		// Consensus rule change deployments.
		//
		// The miner confirmation window is defined as:
		//   target proof of work timespan / target proof of work spacing
		RuleChangeActivationThreshold: 1512, // 75% of MinerConfirmationWindow
		MinerConfirmationWindow:       2016,
		Deployments: [chaincfg.DefinedDeployments]chaincfg.ConsensusDeployment{
			chaincfg.DeploymentTestDummy: {
				BitNumber: 28,
				DeploymentStarter: chaincfg.NewMedianTimeDeploymentStarter(
					time.Unix(1199145601, 0), // January 1, 2008 UTC
				),
				DeploymentEnder: chaincfg.NewMedianTimeDeploymentEnder(
					time.Unix(1230767999, 0), // December 31, 2008 UTC
				),
			},
			chaincfg.DeploymentCSV: {
				BitNumber: 0,
				DeploymentStarter: chaincfg.NewMedianTimeDeploymentStarter(
					time.Unix(1483228800, 0), // January 1, 2017
				),
				DeploymentEnder: chaincfg.NewMedianTimeDeploymentEnder(
					time.Unix(1517356801, 0), // January 31st, 2018
				),
			},
			chaincfg.DeploymentSegwit: {
				BitNumber: 1,
				DeploymentStarter: chaincfg.NewMedianTimeDeploymentStarter(
					time.Unix(1483228800, 0), // January 1, 2017
				),
				DeploymentEnder: chaincfg.NewMedianTimeDeploymentEnder(
					time.Unix(1517356801, 0), // January 31st, 2018
				),
			},
		},

		// Mempool parameters
		RelayNonStdTxs: true,

		// Human-readable part for Bech32 encoded segwit addresses, as defined in
		// BIP 173.
		Bech32HRPSegwit: "tltc", // always tltc for test net

		// Address encoding magics
		PubKeyHashAddrID:        0x6f, // starts with m or n
		ScriptHashAddrID:        0x3a, // starts with Q
		WitnessPubKeyHashAddrID: 0x52, // starts with QW
		WitnessScriptHashAddrID: 0x31, // starts with T7n
		PrivateKeyID:            0xef, // starts with 9 (uncompressed) or c (compressed)

		// BIP32 hierarchical deterministic extended key magics
		HDPrivateKeyID: [4]byte{0x04, 0x35, 0x83, 0x94}, // starts with tprv
		HDPublicKeyID:  [4]byte{0x04, 0x35, 0x87, 0xcf}, // starts with tpub

		// BIP44 coin type used in the hierarchical deterministic path for
		// address generation.
		HDCoinType: 1,
	}

	bigOne = big.NewInt(1)
	// regressionPowLimit is the highest proof of work value a Litecoin block
	// can have for the regression test network.  It is the value 2^255 - 1.
	regressionPowLimit = new(big.Int).Sub(new(big.Int).Lsh(bigOne, 255), bigOne)

	// regTestGenesisHash is the hash of the first block in the block chain for the
	// regression test network (genesis block).
	regTestGenesisHash = chainhash.Hash([chainhash.HashSize]byte{ // Make go vet happy.
		0xf9, 0x16, 0xc4, 0x56, 0xfc, 0x51, 0xdf, 0x62,
		0x78, 0x85, 0xd7, 0xd6, 0x74, 0xed, 0x02, 0xdc,
		0x88, 0xa2, 0x25, 0xad, 0xb3, 0xf0, 0x2a, 0xd1,
		0x3e, 0xb4, 0x93, 0x8f, 0xf3, 0x27, 0x08, 0x53,
	})

	// regTestGenesisMerkleRoot is the hash of the first transaction in the genesis
	// block for the regression test network.  It is the same as the merkle root for
	// the main network.
	regTestGenesisMerkleRoot = genesisMerkleRoot

	// regTestGenesisBlock defines the genesis block of the block chain which serves
	// as the public transaction ledger for the regression test network.
	regTestGenesisBlock = wire.MsgBlock{
		Header: wire.BlockHeader{
			Version:    1,
			PrevBlock:  chainhash.Hash{},         // 0000000000000000000000000000000000000000000000000000000000000000
			MerkleRoot: regTestGenesisMerkleRoot, // 97ddfbbae6be97fd6cdf3e7ca13232a3afff2353e29badfab7f73011edd4ced9
			Timestamp:  time.Unix(1296688602, 0), // 2011-02-02 23:16:42 +0000 UTC
			Bits:       0x207fffff,               // 545259519 [7fffff0000000000000000000000000000000000000000000000000000000000]
			Nonce:      0,
		},
		Transactions: []*wire.MsgTx{&genesisCoinbaseTx},
	}

	ltcRegressionNetParams = &chaincfg.Params{
		Name:        "regtest",
		Net:         wire.TestNet,
		DefaultPort: "18444",
		DNSSeeds:    []chaincfg.DNSSeed{},

		// Chain parameters
		GenesisBlock:             &regTestGenesisBlock,
		GenesisHash:              &regTestGenesisHash,
		PowLimit:                 regressionPowLimit,
		PowLimitBits:             0x207fffff,
		CoinbaseMaturity:         100,
		BIP0034Height:            100000000, // Not active - Permit ver 1 blocks
		BIP0065Height:            1351,      // Used by regression tests
		BIP0066Height:            1251,      // Used by regression tests
		SubsidyReductionInterval: 150,
		TargetTimespan:           time.Hour * 24 * 14, // 14 days
		TargetTimePerBlock:       time.Minute * 10,    // 10 minutes
		RetargetAdjustmentFactor: 4,                   // 25% less, 400% more
		ReduceMinDifficulty:      true,
		MinDiffReductionTime:     time.Minute * 20, // TargetTimePerBlock * 2
		GenerateSupported:        true,

		// Checkpoints ordered from oldest to newest.
		Checkpoints: nil,

		// Consensus rule change deployments.
		//
		// The miner confirmation window is defined as:
		//   target proof of work timespan / target proof of work spacing
		RuleChangeActivationThreshold: 108, // 75%  of MinerConfirmationWindow
		MinerConfirmationWindow:       144,
		Deployments: [chaincfg.DefinedDeployments]chaincfg.ConsensusDeployment{
			chaincfg.DeploymentTestDummy: {
				BitNumber: 28,
				DeploymentStarter: chaincfg.NewMedianTimeDeploymentStarter(
					time.Unix(0, 0), // Always available for vote
				),
				DeploymentEnder: chaincfg.NewMedianTimeDeploymentEnder(
					time.Unix(math.MaxInt64, 0), // Never expires
				),
			},
			chaincfg.DeploymentCSV: {
				BitNumber: 0,
				DeploymentStarter: chaincfg.NewMedianTimeDeploymentStarter(
					time.Unix(0, 0), // Always available for vote
				),
				DeploymentEnder: chaincfg.NewMedianTimeDeploymentEnder(
					time.Unix(math.MaxInt64, 0), // Never expires
				),
			},
			chaincfg.DeploymentSegwit: {
				BitNumber: 1,
				DeploymentStarter: chaincfg.NewMedianTimeDeploymentStarter(
					time.Unix(0, 0), // Always available for vote
				),
				DeploymentEnder: chaincfg.NewMedianTimeDeploymentEnder(
					time.Unix(math.MaxInt64, 0), // Never expires.
				),
			},
		},

		// Mempool parameters
		RelayNonStdTxs: true,

		// Human-readable part for Bech32 encoded segwit addresses, as defined in
		// BIP 173.
		Bech32HRPSegwit: "bcrt", // always bcrt for reg test net

		// Address encoding magics
		PubKeyHashAddrID: 0x6f, // starts with m or n
		ScriptHashAddrID: 0xc4, // starts with 2
		PrivateKeyID:     0xef, // starts with 9 (uncompressed) or c (compressed)

		// BIP32 hierarchical deterministic extended key magics
		HDPrivateKeyID: [4]byte{0x04, 0x35, 0x83, 0x94}, // starts with tprv
		HDPublicKeyID:  [4]byte{0x04, 0x35, 0x87, 0xcf}, // starts with tpub

		// BIP44 coin type used in the hierarchical deterministic path for
		// address generation.
		HDCoinType: 1,
	}
)

func newHashFromStr(hashString string) *chainhash.Hash {
	hash, _ := chainhash.NewHashFromStr(hashString)
	return hash
}

func init() {

	btcMainNetParams.Checkpoints = []chaincfg.Checkpoint{
		{
			Height: 588665,
			Hash:   newHashFromStr("0000000000000000001bbd854842ad2562993e71ae06ed7ecaf8f04f07688692"),
		},
	}

	btcTestNet3Params.Checkpoints = []chaincfg.Checkpoint{
		{
			Height: 1568498,
			Hash:   newHashFromStr("000000000000004654a8d2599a24a95274f9d26c57be147e1c94324071d7363e"),
		},
	}

	ltcMainNetParams.Checkpoints = []chaincfg.Checkpoint{
		{
			Height: 1679794,
			Hash:   newHashFromStr("ca37528e6644cf6a4417493909699881e920086ec609679387c9ee83c73c7ce3"),
		},
	}

	ltcTestNet4Params.Checkpoints = []chaincfg.Checkpoint{
		{
			Height: 1162128,
			Hash:   newHashFromStr("1b0bb65ecba47aa1991cbf1d2adb9348a493040becb871ce1d750e027d900cf4"),
		},
	}

}

func (currency Currency) ChainParam(chainName string) *chaincfg.Params {
	switch currency {
	case Nothing:
		return nil // for genesis blocks
	case Bitcoin:
		if chainName == chain.Testing {
			return btcTestNet3Params
		} else if chainName == chain.Bitmark {
			return btcMainNetParams
		} else {
			return btcRegressionNetParams
		}
	case Litecoin:
		if chainName == chain.Testing {
			return ltcTestNet4Params
		} else if chainName == chain.Bitmark {
			return ltcMainNetParams
		} else {
			return ltcRegressionNetParams
		}
	default:
		logger.Panicf("non supported currency: %s", currency)
	}
	return nil
}
