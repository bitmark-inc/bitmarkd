// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// Package peer - this module handles the peer to peer network
//
// server-side:
//
// * upstream sending of block, transactions
// * listener for RPC requests e.g. retrieve old block
//
// client-side
//
// * connector to retrieve missing data from other listeners
package peer
