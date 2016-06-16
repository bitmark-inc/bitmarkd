// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// maintain the on-disk data store
//
//
// maintain separate pools of a number of elements in key->value form
//
// This maintains a LevelDB database split into a series of tables.
// Each table is defined by a prefix byte that is obtainen from the
// prefix tag in the struct defining the avaiable tables.
//
//
// Notes:
// 1. each separate pool has a single byte prefix (to spread the keys in LevelDB)
// 2. digest = SHA3-256(data)
//
// Blocks:
//
//   B<block-number>                - block store (already mined blocks)
//                                    data: header ++ count ++ merkle tree of transaction digests
//                                    the transactions must be in the mined transactions table
//   F<block-number>                - current block owner
//                                    data: account ++ currency ++ address
//
// Transactions:
//
//   T<tx-digest>                   - mined transactions: packed transaction data
//   V<tx-digest>                   - verified transactions: packed transaction data
//
// Assets:
//
//   I<assetIndex>                  - transaction-digest (to locate the AssetData transaction)
//
// Ownership:
//
//   N<owner-pubkey>                - count (for owner indexing)
//   K<owner-pubkey><count>         - tx-digest ++ issue tx-digest ++ asset-digest
//   D<owner-pubkey><tx-digest>     - count
//
// Networking:
//
//   P<IP:port>                     - P2P: ZMQ public-key
//   R<IP:port>                     - RPC: certificate-fingerprint
//   C<fingerprint>                 - raw certificate
//
package storage
