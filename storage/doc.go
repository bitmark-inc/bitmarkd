// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// Package storage - maintain the on-disk data store
//
//
// maintain separate pools of a number of elements in key->value form
//
// This maintains a LevelDB database split into a series of tables.
// Each table is defined by a prefix byte that is obtained from the
// prefix tag in the struct defining the available tables.
//
//
// Notes:
// 1. each separate pool has a single byte prefix (to spread the keys in LevelDB)
// 2. ⧺            = concatenation of byte data
// 3. BN           = block number as 8 byte big endian (uint64)
// 4. txId         = transaction digest as 32 byte SHA3-256(data)
// 5. asset id     = fingerprint digest as 64 byte SHA3-512(data)
// 6. count        = successive index value as 8 byte big endian (uint64)
// 7. owner        = bitmark account (prefix ⧺ public key ≡ 33 bytes if Ed25519)
// 8. 00           = single byte values 00..ff
// 9. value        = balance quantity value as 8 byte big endian (uint64)
//10. *others*     = byte values of various length
//
// Blocks:
//
//   B ⧺ BN               - block store
//                          data: header ⧺ (concat transactions)
//   2 ⧺ BN               - block Argon2 hashes
//                          data: hash of block
//   H ⧺ BN               - current block currencies
//                          data: map(currency → currency address)
//   I ⧺ txId             - current block owner transaction index
//                          data: BN
//
//
// Transactions:
//
//   T ⧺ txId             - confirmed transactions
//                          data: BN ⧺ packed transaction data
//
// Assets:
//
//   A ⧺ asset id         - confirmed asset identifier
//                          data: BN ⧺ packed asset data
//
//
// Ownership:
//
//   N ⧺ owner            - next count value to use for appending to owned items
//                          data: count
//   L ⧺ owner ⧺ count    - list of owned items
//                          data: txId
//   D ⧺ owner ⧺ txId     - position in list of owned items, for delete after transfer
//                          data: count
//   P ⧺ txId             - owner data (00=asset, 01=block, 02=share) head of provenance chain
//                          data: 00 ⧺ transfer BN ⧺ issue txId ⧺ issue BN ⧺ asset id
//                          data: 01 ⧺ transfer BN ⧺ issue txId ⧺ issue BN ⧺ owned BN
//                          data: 02 ⧺ transfer BN ⧺ issue txId ⧺ issue BN ⧺ asset id
//
//
// Bitmark Shares (txId ≡ share id)
//
//   F ⧺ txId             - share total value (constant)
//                          data: value ⧺ txId
//   Q ⧺ owner ⧺ txId     - current balance quantity of shares (ShareId) for each owner (deleted if value becomes zero)
//                          data: value
//
// Testing:
//   Z ⧺ key              - testing data
package storage
