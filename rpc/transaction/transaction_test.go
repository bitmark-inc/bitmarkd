// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package transaction_test

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/bitmarkd/rpc/fixtures"
	"github.com/bitmark-inc/bitmarkd/rpc/mocks"
	"github.com/bitmark-inc/bitmarkd/rpc/transaction"
	"github.com/bitmark-inc/logger"
)

func TestTransactionStatus(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	r := mocks.NewMockReservoir(ctl)

	now := time.Now()

	tr := transaction.New(logger.New(fixtures.LogCategory), now, r)

	arg := transaction.Arguments{TxId: merkle.Digest{1, 2, 3, 4}}

	r.EXPECT().TransactionStatus(arg.TxId).Return(reservoir.StateConfirmed).Times(1)

	var reply transaction.StatusReply
	err := tr.Status(&arg, &reply)
	assert.Nil(t, err, "wrong Status")
	assert.Equal(t, reservoir.StateConfirmed.String(), reply.Status, "")
}

func TestTransactionStatusWhenReservoirEmpty(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	now := time.Now()

	tr := transaction.New(logger.New(fixtures.LogCategory), now, nil)

	arg := transaction.Arguments{TxId: merkle.Digest{1, 2, 3, 4}}

	var reply transaction.StatusReply
	err := tr.Status(&arg, &reply)
	assert.NotNil(t, err, "wrong Status")
	assert.Equal(t, fault.MissingReservoir, err, "wrong error message")
}
