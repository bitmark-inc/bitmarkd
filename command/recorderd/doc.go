// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// Proof-of-work program for bitmark system
//
// This program subscribes to potential blocks stream on a bitmarkd
// and determines an argon2 hash value that meets the current network
// difficulty value.
package main
