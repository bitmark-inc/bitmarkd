// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// Package asset - Cache for assets
//
// temporary store assets just received until they are:
// a. verified by having a corresponding issue verified
// b. confirmed by having a block broadcast containing them
// c. expired because no longer referenced by any issues
//    i.e. all isues were expired
package asset
