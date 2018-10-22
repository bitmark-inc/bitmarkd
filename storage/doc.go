// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// maintain the on-disk data store
//
//
// maintain separate pools of a number of elements in key->value form
//
// This maintains a LevelDB database split into a series of tables.
// Each table is defined by a prefix byte that is obtained from the
// prefix tag in the struct defining the avaiable tables.
//
//
// Notes:
// 1. each separate pool has a single byte prefix (to spread the keys in LevelDB)
// 2. ++           = concatenation of byte data
// 3. BN           = block number as 8 byte big endian (uint64)
// 4. txId         = transaction digest as 32 byte SHA3-256(data)
// 5. asset id     = fingerprint digest as 64 byte SHA3-512(data)
// 6. count        = successive index value as 8 byte big endian (uint64)
// 7. owner        = bitmark account (32 byte public key)
// 8. 00           = single byte values 00..ff
// 9. *others*     = byte values of various length
//
// Blocks:
//
//   B ++ BN               - block store
//                           data: header ++ (concat transactions)
//   H ++ BN               - current block currencies
//                           data: map(currency → currency address)
//   I ++ txId             - current block owner transaction index
//                           data: BN
//
//
// Transactions:
//
//   T ++ txId             - confirmed transactions
//                           data: BN ++ packed transaction data
//
// Assets:
//
//   A ++ asset id         - confirmed asset identifier
//                           data: BN ++ packed asset data
//
// Ownership:
//
//   N ++ owner            - next count value to use for appending to owned items
//                           data: count
//   K ++ owner ++ count   - list of owned items
//                           data: 00 ++ last transfer txId ++ last transfer BN ++ issue txId ++ issue BN ++ asset id
//                           data: 01 ++ last transfer txId ++ last transfer BN ++ issue txId ++ issue BN ++ owned BN
//   D ++ owner ++ txId    - position in list of owned items, for delete after transfer
//                           data: count
//
// Testing:
//   Z ++ key              - testing data
package storage
