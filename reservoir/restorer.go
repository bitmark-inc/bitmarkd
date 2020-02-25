// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package reservoir

import (
	"fmt"

	"github.com/bitmark-inc/bitmarkd/asset"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
)

// Restorer - interface to restore data from cache file
type Restorer interface {
	Restore() error
	fmt.Stringer
}

// NewTransactionRestorer - create object with Restorer interface
func NewTransactionRestorer(unpacked interface{}, packed interface{}, handles Handles) (Restorer, error) {
	switch t := unpacked.(type) {
	case *transactionrecord.BitmarkIssue:

		return &issueRestoreData{
			packed:                  packed.(transactionrecord.Packed),
			assetHandle:             handles.Assets,
			blockOwnerPaymentHandle: handles.BlockOwnerPayment,
		}, nil

	case *transactionrecord.AssetData:

		return &assetRestoreData{unpacked: t}, nil

	case *transactionrecord.BitmarkTransferUnratified,
		*transactionrecord.BitmarkTransferCountersigned,
		*transactionrecord.BitmarkShare:

		return &transferRestoreData{
			unpacked:          unpacked.(transactionrecord.BitmarkTransfer),
			transaction:       handles.Transactions,
			ownerTx:           handles.OwnerTx,
			ownerData:         handles.OwnerData,
			blockOwnerPayment: handles.BlockOwnerPayment,
		}, nil

	case *transactionrecord.ShareGrant:

		return &grantRestoreData{
			unpacked:          t,
			shareQuantity:     handles.ShareQuantity,
			share:             handles.Share,
			ownerData:         handles.OwnerData,
			blockOwnerPayment: handles.BlockOwnerPayment,
		}, nil

	case *transactionrecord.ShareSwap:

		return &swapRestoreData{
			unpacked:          t,
			shareQuantity:     handles.ShareQuantity,
			share:             handles.Share,
			ownerData:         handles.OwnerData,
			blockOwnerPayment: handles.BlockOwnerPayment,
		}, nil

	default:
		return nil, fmt.Errorf("unhandled restore tx type: %d", t)
	}
	//panic("cannot get here")
}

type assetRestoreData struct {
	unpacked *transactionrecord.AssetData
}

func (a *assetRestoreData) String() string {
	return "transactionrecord.AssetData"
}

func (a *assetRestoreData) Restore() error {
	_, _, err := asset.Cache(a.unpacked, storage.Pool.Assets)
	if nil != err {
		return fmt.Errorf("fail to cache asset: %s", err)
	}
	return nil
}

type issueRestoreData struct {
	packed                  transactionrecord.Packed
	assetHandle             storage.Handle
	blockOwnerPaymentHandle storage.Handle
}

func (i *issueRestoreData) String() string {
	return "transactionrecord.BitmarkIssue"
}

func (i *issueRestoreData) Restore() error {
	issues := make([]*transactionrecord.BitmarkIssue, 0, 100)

	for len(i.packed) > 0 {
		transaction, n, err := i.packed.Unpack(mode.IsTesting())
		if nil != err {
			return fmt.Errorf("unable to unpack issue: %s", err)
		}

		if issue, ok := transaction.(*transactionrecord.BitmarkIssue); ok {
			issues = append(issues, issue)
		} else {
			return fmt.Errorf("issue block contains non-issue: %+v", transaction)
		}
		i.packed = i.packed[n:]
	}

	_, _, err := storeIssues(issues, i.assetHandle, i.blockOwnerPaymentHandle)
	if nil != err {
		return fmt.Errorf("store issue with error: %s", err)
	}

	return nil
}

type transferRestoreData struct {
	unpacked          transactionrecord.BitmarkTransfer
	transaction       storage.Handle
	ownerTx           storage.Handle
	ownerData         storage.Handle
	blockOwnerPayment storage.Handle
}

func (t *transferRestoreData) String() string {
	return "transactionrecord.BitmarkTransfer"
}

func (t *transferRestoreData) Restore() error {
	_, _, err := storeTransfer(t.unpacked, t.transaction, t.ownerTx, t.ownerData, t.blockOwnerPayment)
	if nil != err {
		return fmt.Errorf("fail to restore transfer: %s", err)
	}
	return err
}

type grantRestoreData struct {
	unpacked          *transactionrecord.ShareGrant
	shareQuantity     storage.Handle
	share             storage.Handle
	ownerData         storage.Handle
	blockOwnerPayment storage.Handle
}

func (g *grantRestoreData) String() string {
	return "transactionrecord.ShareGrant"
}

func (g *grantRestoreData) Restore() error {
	_, _, err := storeGrant(g.unpacked, g.shareQuantity, g.share, g.ownerData, g.blockOwnerPayment, storage.Pool.Transactions)

	if nil != err {
		return fmt.Errorf("fail to restore grant: %s", err)
	}
	return err
}

type swapRestoreData struct {
	unpacked          *transactionrecord.ShareSwap
	shareQuantity     storage.Handle
	share             storage.Handle
	ownerData         storage.Handle
	blockOwnerPayment storage.Handle
}

func (s *swapRestoreData) String() string {
	return "transactionrecord.ShareSwap"
}

func (s *swapRestoreData) Restore() error {
	_, _, err := storeSwap(s.unpacked, s.shareQuantity, s.share, s.ownerData, s.blockOwnerPayment)
	if nil != err {
		return fmt.Errorf("create swap restorer with error: %s", err)
	}
	return err
}
