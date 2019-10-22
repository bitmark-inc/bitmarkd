package reservoir

import (
	"fmt"

	"github.com/bitmark-inc/bitmarkd/asset"

	"github.com/bitmark-inc/bitmarkd/storage"

	"github.com/prometheus/common/log"

	"github.com/bitmark-inc/bitmarkd/mode"

	"github.com/bitmark-inc/bitmarkd/transactionrecord"
)

// Restorer - interface to restore data from cache file
type Restorer interface {
	Restore() error
}

// NewRestorer - create object with Restorer interface
func NewRestorer(t interface{}, args ...interface{}) (Restorer, error) {
	switch t.(type) {
	case *transactionrecord.BitmarkIssue:
		if 3 != len(args) {
			return nil, fmt.Errorf("insufficient parameter")
		}
		return &issueRestoreData{
			packed:                  args[0].(transactionrecord.Packed),
			assetHandle:             args[1].(storage.Handle),
			blockOwnerPaymentHandle: args[2].(storage.Handle),
		}, nil

	case *transactionrecord.AssetData:
		return &assetRestoreData{packed: t.(*transactionrecord.AssetData)}, nil

	case *transactionrecord.BitmarkTransferUnratified,
		*transactionrecord.BitmarkTransferCountersigned:

		if 4 != len(args) {
			return nil, fmt.Errorf("insufficient parameter")
		}

		return &transferRestoreData{
			packed:            t.(transactionrecord.BitmarkTransfer),
			transaction:       args[0].(storage.Handle),
			ownerTx:           args[1].(storage.Handle),
			ownerData:         args[2].(storage.Handle),
			blockOwnerPayment: args[3].(storage.Handle),
		}, nil
	}
	return nil, nil
}

type assetRestoreData struct {
	packed *transactionrecord.AssetData
}

func (a *assetRestoreData) Restore() error {
	_, _, err := asset.Cache(a.packed, storage.Pool.Assets)
	if nil != err {
		msg := fmt.Errorf("fail to cache asset: %s", err)
		log.Errorf("%s", msg)
		return msg
	}
	return nil
}

type issueRestoreData struct {
	packed                  transactionrecord.Packed
	assetHandle             storage.Handle
	blockOwnerPaymentHandle storage.Handle
}

func (i *issueRestoreData) Restore() error {
	issues := make([]*transactionrecord.BitmarkIssue, 0, 100)

	for len(i.packed) > 0 {
		transaction, n, err := i.packed.Unpack(mode.IsTesting())
		if nil != err {
			msg := fmt.Errorf("unable to unpack issue: %s", err)
			log.Errorf("%s", msg)
			return msg
		}

		if issue, ok := transaction.(*transactionrecord.BitmarkIssue); ok {
			issues = append(issues, issue)
		} else {
			msg := fmt.Errorf("issue block contains non-issue: %+v", transaction)
			log.Errorf("%s", msg)
			return msg
		}
		i.packed = i.packed[n:]
	}

	_, _, err := StoreIssues(issues, i.assetHandle, i.blockOwnerPaymentHandle)
	if nil != err {
		log.Errorf("fail to store issue: %s", err)
	}

	return nil
}

type transferRestoreData struct {
	packed            transactionrecord.BitmarkTransfer
	transaction       storage.Handle
	ownerTx           storage.Handle
	ownerData         storage.Handle
	blockOwnerPayment storage.Handle
}

func (t *transferRestoreData) Restore() error {
	_, _, err := StoreTransfer(t.packed, t.transaction, t.ownerTx, t.ownerData, t.blockOwnerPayment)
	if nil != err {
		log.Errorf("fail to restore transfer: %s", err)
	}
	return nil
}
