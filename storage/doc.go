// Copyright (c) 2014-2017 Bitmark Inc.
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
// 3. block number = big endian uint64 (8 bytes)
// 4. txId         = transaction digest as 32 byte SHA3-256(data)
// 5. asset index  = fingerprint digest as 64 byte SHA3-512(data)
// 6. count        = successive index value as big endian uint64 (8 bytes)
// 7. owner        = bitmark account (32 byte public key)
// 8. *others*     = byte values of various length
//
// Blocks:
//
//   B ++ block number          - block store
//                                data: header ++ base transaction ++ (concat transactions)
//   F ++ block number          - current block owner
//                                data: owner ++ currency ++ currency address
//
// Transactions:
//
//   T ++ txId                  - confirmed transactions
//                                data: packed transaction data
//
// Assets:
//
//   A ++ asset index           - confirmed asset
//                                data: packed asset data
//
// Ownership:
//
//   N ++ owner                 - next count value to use for appending to owned items
//                                data: count
//   K ++ owner ++ count        - list of owned items
//                                data: last transfer txId ++ last transfer block number ++ issue txId ++ issue block number ++ asset index
//   D ++ owner ++ txId         - position in list of owned items, for delete after transfer
//                                data: count
//
// Payment:
//
//   C ++ currency(uint64)      - currency processing
//                                data: latest block number (big endian uint64, 8 bytes)
//
//   P ++ payId                 - payment confirmation (array of addresses + values)
//                                data: currency(varint) ++ txId_bytes(varint) ++ txId ++ [ count(varint) ++ address ++ value(varint) ]
//
//
// Testing:
//   Z ++ key                   - testing data
package storage
