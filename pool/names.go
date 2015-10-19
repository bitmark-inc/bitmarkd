// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package pool

// Notes:
// 1. each separate pool has a single byte prefix (to spread the keys in LevelDB)
// 2. digest = sha256(sha256(data)) i.e. must be compatible with Bitcoin merkle tree
// 3. prop = registration-transfer-digest
//
// Blocks:
//
//   B<block-number>       - block store (already mined blocks) = header + cbLength + coinbase + count + merkle tree of transactions
//
// Transactions:
//
//   T<tx-digest>          - packed transaction data
//   S<tx-digest>          - state: byte[expired(E), pending(P), verified(V), confirmed(C)] ++ int64[the U/V table count value]
//   U<count>              - transaction-digest ++ int64[timestamp] (pending unverified transactions waiting for payment)
//   V<count>              - transaction-digest ++ int64[timestamp] (verified transactions, Available for mining)
//
// Assets:
//
//   I<assetIndex>         - transaction-digest (to locate the AssetData transaction)
//
// Ownership:
//
//   O<bmtran-digest>      - owner public key ++ registration digest (to check current ownership of property)
//
// Networking:
//
//   P<IP:port>            - P2P: ZMQ public-key
//   R<IP:port>            - RPC: certificate-fingerprint
//   C<fingerprint>        - raw certificate

// type for pool name
type nameb byte

// Names of the pools
const (
	// networking pools
	Peers        = nameb('P')
	RPCs         = nameb('R')
	Certificates = nameb('C')

	// transaction data pools
	TransactionData  = nameb('T')
	TransactionState = nameb('S')

	// transaction index pools
	PendingIndex  = nameb('U')
	VerifiedIndex = nameb('V')

	// asset
	AssetData = nameb('I')

	// ownership indexes
	OwnerIndex = nameb('O')

	// blocks
	BlockData = nameb('B')

	// just for testing
	TestData = nameb('Z')
)
