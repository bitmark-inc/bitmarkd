// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package transactionrecord

import (
	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/merkle"
)

// for unratified

func (transfer *BitmarkTransferUnratified) GetLink() merkle.Digest {
	return transfer.Link
}

func (transfer *BitmarkTransferUnratified) GetPayment() *Payment {
	return transfer.Escrow
}

func (transfer *BitmarkTransferUnratified) GetOwner() *account.Account {
	return transfer.Owner
}

func (transfer *BitmarkTransferUnratified) GetCurrencies() currency.Map {
	return nil
}

func (transfer *BitmarkTransferUnratified) GetSignature() account.Signature {
	return transfer.Signature
}

func (transfer *BitmarkTransferUnratified) GetCountersignature() account.Signature {
	return nil
}

// for countersigned

func (transfer *BitmarkTransferCountersigned) GetLink() merkle.Digest {
	return transfer.Link
}

func (transfer *BitmarkTransferCountersigned) GetPayment() *Payment {
	return transfer.Escrow
}

func (transfer *BitmarkTransferCountersigned) GetOwner() *account.Account {
	return transfer.Owner
}

func (transfer *BitmarkTransferCountersigned) GetCurrencies() currency.Map {
	return nil
}

func (transfer *BitmarkTransferCountersigned) GetSignature() account.Signature {
	return transfer.Signature
}

func (transfer *BitmarkTransferCountersigned) GetCountersignature() account.Signature {
	return transfer.Countersignature
}

// for block owner transfer

func (transfer *BlockOwnerTransfer) GetLink() merkle.Digest {
	return transfer.Link
}

func (transfer *BlockOwnerTransfer) GetPayment() *Payment {
	return transfer.Escrow
}

func (transfer *BlockOwnerTransfer) GetOwner() *account.Account {
	return transfer.Owner
}

func (transfer *BlockOwnerTransfer) GetCurrencies() currency.Map {
	return transfer.Payments
}

func (transfer *BlockOwnerTransfer) GetSignature() account.Signature {
	return transfer.Signature
}

func (transfer *BlockOwnerTransfer) GetCountersignature() account.Signature {
	return transfer.Countersignature
}

// for share

func (share *BitmarkShare) GetLink() merkle.Digest {
	return share.Link
}

func (share *BitmarkShare) GetPayment() *Payment {
	return nil
}

func (share *BitmarkShare) GetOwner() *account.Account {
	return nil
}

func (share *BitmarkShare) GetCurrencies() currency.Map {
	return nil
}

func (share *BitmarkShare) GetSignature() account.Signature {
	return share.Signature
}

func (share *BitmarkShare) GetCountersignature() account.Signature {
	return nil
}
