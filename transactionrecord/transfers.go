// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package transactionrecord

import (
	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/merkle"
)

// for unratified

func (transfer *BitmarkTransferUnratified) GetLink() merkle.Digest {
	return transfer.Link
}

func (transfer *BitmarkTransferUnratified) GetPayment() *Payment {
	return transfer.Payment
}

func (transfer *BitmarkTransferUnratified) GetOwner() *account.Account {
	return transfer.Owner
}

func (transfer *BitmarkTransferUnratified) GetSignature() account.Signature {
	return transfer.Signature
}

func (transfer *BitmarkTransferUnratified) GetCountersignature() account.Signature {
	return account.Signature{}
}

// for countersigned

func (transfer *BitmarkTransferCountersigned) GetLink() merkle.Digest {
	return transfer.Link
}

func (transfer *BitmarkTransferCountersigned) GetPayment() *Payment {
	return transfer.Payment
}

func (transfer *BitmarkTransferCountersigned) GetOwner() *account.Account {
	return transfer.Owner
}

func (transfer *BitmarkTransferCountersigned) GetSignature() account.Signature {
	return transfer.Signature
}

func (transfer *BitmarkTransferCountersigned) GetCountersignature() account.Signature {
	return transfer.Countersignature
}
