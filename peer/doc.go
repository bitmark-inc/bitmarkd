// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// this module handle the peer to peer network
//
// server-side:
//
// * broadcaster of block, transactions
// * listener for RPC requests e.g. retrieve old block
//
// client-side
//
// * subscriber listens to several broadcasters
// * connector to retrieve missing data from other listeners
package peer
