// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpccalls

import (
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/rpc/transaction"
)

// TransactionStatusData - request data fro transaction status
type TransactionStatusData struct {
	TxId string
}

// GetTransactionStatus - perform a status request
func (client *Client) GetTransactionStatus(statusConfig *TransactionStatusData) (*transaction.StatusReply, error) {

	var txId merkle.Digest
	err := txId.UnmarshalText([]byte(statusConfig.TxId))
	if nil != err {
		return nil, err
	}

	statusArgs := transaction.Arguments{
		TxId: txId,
	}

	client.printJson("Status Request", statusArgs)

	var reply transaction.StatusReply
	err = client.client.Call("Transaction.Status", statusArgs, &reply)
	if err != nil {
		return nil, err
	}

	client.printJson("Status Reply", reply)

	return &reply, nil
}
