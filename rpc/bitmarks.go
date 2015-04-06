// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"github.com/bitmark-inc/bitmarkd/transaction"
	"github.com/bitmark-inc/logger"
)

// Bitmarks
// --------

type Bitmarks struct {
	bitmark *Bitmark
	log     *logger.L
}

// Bitmarks issue
// --------------

func (bitmarks *Bitmarks) Issue(arguments *[]transaction.BitmarkIssue, reply *[]BitmarkIssueReply) error {

	bitmark := bitmarks.bitmark

	result := make([]BitmarkIssueReply, len(*arguments))
	for i, argument := range *arguments {
		if err := bitmark.Issue(&argument, &result[i]); err != nil {
			result[i].Err = err.Error()
		}
	}

	*reply = result
	return nil
}

// Bitmarks transfer
// -----------------

func (bitmarks *Bitmarks) Transfer(arguments *[]transaction.BitmarkTransfer, reply *[]BitmarkTransferReply) error {

	bitmark := bitmarks.bitmark

	result := make([]BitmarkTransferReply, len(*arguments))
	for i, argument := range *arguments {
		if err := bitmark.Transfer(&argument, &result[i]); err != nil {
			result[i].Err = err.Error()
		}
	}

	*reply = result
	return nil
}
