// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package reservoir

import (
	"bytes"
	"encoding/binary"

	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/logger"
)

// BalanceInfo - result returned by store share
type BalanceInfo struct {
	ShareId   merkle.Digest `json:"shareId"`
	Confirmed uint64        `json:"confirmed"`
	Spend     uint64        `json:"spend"`
	Available uint64        `json:"available"`
}

// ShareBalance - get a list of balances
func ShareBalance(owner *account.Account, startShareId merkle.Digest, count int) ([]BalanceInfo, error) {

	ownerBytes := owner.Bytes()
	prefix := append(ownerBytes, startShareId[:]...)

	cursor := storage.Pool.ShareQuantity.NewFetchCursor().Seek(prefix)

	items, err := cursor.Fetch(count)
	if nil != err {
		return nil, err
	}

	records := make([]BalanceInfo, 0, len(items))

loop:
	for _, item := range items {
		n := len(item.Key)
		split := n - len(startShareId)
		if split <= 0 {
			logger.Panicf("split cannot be <= 0: %d", split)
		}
		itemOwner := item.Key[:n-len(startShareId)]
		if !bytes.Equal(ownerBytes, itemOwner) {
			break loop
		}

		value := binary.BigEndian.Uint64(item.Value[:8])

		var shareId merkle.Digest
		copy(shareId[:], item.Key[split:])

		spendKey := makeSpendKey(owner, shareId)

		globalData.RLock()
		spend := globalData.spend[spendKey]
		globalData.RUnlock()

		records = append(records, BalanceInfo{
			ShareId:   shareId,
			Confirmed: value,
			Spend:     spend,
			Available: value - spend,
		})
	}

	return records, nil
}
