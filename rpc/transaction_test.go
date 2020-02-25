// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc_test

import (
	"testing"
	"time"

	"github.com/bitmark-inc/bitmarkd/reservoir"

	"github.com/bitmark-inc/bitmarkd/rpc/mocks"

	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/bitmarkd/merkle"

	"github.com/bitmark-inc/logger"
	"golang.org/x/time/rate"

	"github.com/bitmark-inc/bitmarkd/rpc"
)

func TestTransaction_Status(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	r := mocks.NewMockReservoir(ctl)

	now := time.Now()

	tr := rpc.Transaction{
		Log:     logger.New(logCategory),
		Limiter: rate.NewLimiter(100, 100),
		Start:   now,
		Rsvr:    r,
	}

	arg := rpc.TransactionArguments{TxId: merkle.Digest{1, 2, 3, 4}}

	r.EXPECT().TransactionStatus(arg.TxId).Return(reservoir.StateConfirmed).Times(1)

	var reply rpc.TransactionStatusReply
	err := tr.Status(&arg, &reply)
	assert.Nil(t, err, "wrong Status")
	assert.Equal(t, reservoir.StateConfirmed.String(), reply.Status, "")
}
