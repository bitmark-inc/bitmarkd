// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package ownership

import (
	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/storage"
)

// Ownership - interface for ownership
type Ownership interface {
	ListBitmarksFor(*account.Account, uint64, int) ([]Record, error)
}

type ownership struct {
	PoolOwnerList storage.Handle
	PoolOwnerData storage.Handle
}

func (o ownership) ListBitmarksFor(owner *account.Account, start uint64, count int) ([]Record, error) {
	return listBitmarksFor(owner, start, count)
}

var data ownership

// Initialise - initialise ownership
func Initialise(ownerList, ownerData storage.Handle) {
	data = ownership{
		PoolOwnerList: ownerList,
		PoolOwnerData: ownerData,
	}
}

// Get - return Record interface
func Get() Ownership {
	return &data
}
