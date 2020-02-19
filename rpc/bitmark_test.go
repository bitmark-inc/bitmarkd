// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc_test

import (
	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/chain"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/bitmarkd/rpc"
	"github.com/bitmark-inc/bitmarkd/rpc/mocks"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
	"testing"
)

func TestBitmarkTransfer(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()

	mode.Initialise(chain.Testing)
	defer mode.Finalise()

	bus := messagebus.Bus.P2P.Chan()
	defer messagebus.Bus.P2P.Release()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	owner := account.Account{
		AccountInterface: &account.ED25519Account{
			Test:      true,
			PublicKey: issuerPublicKey,
		},
	}

	transfer := transactionrecord.BitmarkTransferCountersigned{
		Link:             merkle.Digest{},
		Escrow:           nil,
		Owner:            &owner,
		Signature:        nil,
		Countersignature: nil,
	}

	unratitifed := transactionrecord.BitmarkTransferUnratified{
		Link:      merkle.Digest{},
		Escrow:    nil,
		Owner:     &owner,
		Signature: nil,
	}

	info := reservoir.TransferInfo{
		Id:        pay.PayId{1, 2},
		TxId:      merkle.Digest{1, 2},
		IssueTxId: merkle.Digest{1, 2},
		Packed:    nil,
		Payments:  nil,
	}

	r := mocks.NewMockReservoir(ctl)
	r.EXPECT().StoreTransfer(&unratitifed).Return(&info, false, nil).Times(1)

	b := rpc.Bitmark{
		Log:            logger.New(logCategory),
		Limiter:        rate.NewLimiter(100, 100),
		IsNormalMode:   func(_ mode.Mode) bool { return true },
		IsTestingChain: func() bool { return true },
		Rsvr:           r,
	}
	var reply rpc.BitmarkTransferReply
	err := b.Transfer(&transfer, &reply)
	assert.Nil(t, err, "wrong transfer")
	assert.Equal(t, info.Id, reply.PayId, "wrong payID")
	assert.Equal(t, info.TxId, reply.TxId, "wrong txID")
	assert.Equal(t, 0, len(reply.Payments), "wrong payment count")

	received := <-bus
	assert.Equal(t, "transfer", received.Command, "wrong message")
}
