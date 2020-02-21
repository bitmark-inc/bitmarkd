// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc_test

import (
	"encoding/binary"
	"testing"

	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"

	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/merkle"

	"github.com/bitmark-inc/bitmarkd/rpc/mocks"
	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/bitmarkd/rpc"
	"github.com/bitmark-inc/logger"
	"golang.org/x/time/rate"
)

func TestBlockOwnerTxIdForBlock(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	p := mocks.NewMockHandle(ctl)
	br := mocks.NewMockRecord(ctl)

	b := rpc.BlockOwner{
		Log:     logger.New(logCategory),
		Limiter: rate.NewLimiter(100, 100),
		Pool:    p,
		Br:      br,
	}

	blockNumber := uint64(100)
	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, blockNumber)

	arg := rpc.TxIdForBlockArguments{
		BlockNumber: blockNumber,
	}

	h := blockrecord.Header{
		Version:          1,
		TransactionCount: 2,
		Number:           3,
		PreviousBlock:    blockdigest.Digest{},
		MerkleRoot:       merkle.Digest{},
		Timestamp:        4,
		Difficulty:       nil,
		Nonce:            5,
	}

	d := blockdigest.Digest{}

	p.EXPECT().Get(key).Return([]byte{}).Times(1)
	br.EXPECT().ExtractHeader([]byte{}, uint64(0), false).Return(&h, d, []byte{}, nil).Times(1)

	var reply rpc.TxIdForBlockReply
	err := b.TxIdForBlock(&arg, &reply)
	assert.Nil(t, err, "wrong TxIdForBlock")
	assert.Equal(t, blockrecord.FoundationTxId(3, blockdigest.Digest{}), reply.TxId, "wrong tx ID")
}

func TestBlockOwnerTransfer(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	bus := messagebus.Bus.P2P.Chan()
	defer messagebus.Bus.P2P.Release()

	p := mocks.NewMockHandle(ctl)
	br := mocks.NewMockRecord(ctl)
	r := mocks.NewMockReservoir(ctl)

	b := rpc.BlockOwner{
		Log:            logger.New(logCategory),
		Limiter:        rate.NewLimiter(100, 100),
		Pool:           p,
		Br:             br,
		IsNormalMode:   func(_ mode.Mode) bool { return true },
		IsTestingChain: func() bool { return true },
		Rsvr:           r,
	}

	acc := account.Account{
		AccountInterface: &account.ED25519Account{
			Test:      true,
			PublicKey: issuerPublicKey,
		},
	}

	arg := transactionrecord.BlockOwnerTransfer{
		Link:             merkle.Digest{},
		Escrow:           nil,
		Version:          0,
		Payments:         nil,
		Owner:            &acc,
		Signature:        nil,
		Countersignature: nil,
	}

	info := reservoir.TransferInfo{
		Id:        pay.PayId{1, 2, 3, 4},
		TxId:      merkle.Digest{5, 6, 7, 8},
		IssueTxId: merkle.Digest{2, 4, 6, 8},
		Packed:    nil,
		Payments: []transactionrecord.PaymentAlternative{
			{
				&transactionrecord.Payment{
					Currency: currency.Litecoin,
					Address:  litecoinAddress,
					Amount:   100,
				},
			},
		},
	}

	r.EXPECT().StoreTransfer(&arg).Return(&info, false, nil).Times(1)

	var reply rpc.BlockOwnerTransferReply
	err := b.Transfer(&arg, &reply)
	assert.Nil(t, err, "wrong Transfer")
	assert.Equal(t, info.TxId, reply.TxId, "wrong tx ID")
	assert.Equal(t, info.Id, reply.PayId, "wrong pay ID")
	assert.Equal(t, info.Payments[0], reply.Payments[currency.Litecoin.String()], "wrong payments")

	msg := <-bus
	assert.Equal(t, "transfer", msg.Command, "wrong message command")
}
