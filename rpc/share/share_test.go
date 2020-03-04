// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package share_test

import (
	"crypto/ed25519"
	"testing"

	"github.com/bitmark-inc/bitmarkd/rpc/fixtures"

	"github.com/bitmark-inc/bitmarkd/messagebus"

	"github.com/bitmark-inc/bitmarkd/chain"

	"github.com/bitmark-inc/bitmarkd/currency"

	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/reservoir"

	"github.com/bitmark-inc/bitmarkd/rpc/mocks"
	"github.com/golang/mock/gomock"

	"github.com/bitmark-inc/bitmarkd/mode"

	"github.com/bitmark-inc/bitmarkd/account"

	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/bitmarkd/rpc/share"
	"github.com/bitmark-inc/logger"
)

func TestShareCreate(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	r := mocks.NewMockReservoir(ctl)

	s := share.New(
		logger.New(fixtures.LogCategory),
		func(_ mode.Mode) bool { return true },
		r,
	)

	acc := account.Account{
		AccountInterface: &account.ED25519Account{
			Test:      true,
			PublicKey: fixtures.IssuerPublicKey,
		},
	}

	arg := transactionrecord.BitmarkShare{
		Link:      merkle.Digest{},
		Quantity:  50,
		Signature: nil,
	}

	packed, _ := arg.Pack(&acc)
	arg.Signature = ed25519.Sign(fixtures.IssuerPrivateKey, packed)

	info := reservoir.TransferInfo{
		Id:        pay.PayId{1, 2, 3, 4},
		TxId:      merkle.Digest{5, 6, 7, 8},
		IssueTxId: merkle.Digest{9, 9, 9, 9},
		Packed:    nil,
		Payments: []transactionrecord.PaymentAlternative{
			[]*transactionrecord.Payment{
				{
					Currency: currency.Litecoin,
					Address:  fixtures.LitecoinAddress,
					Amount:   100,
				},
			},
		},
	}

	r.EXPECT().StoreTransfer(&arg).Return(&info, false, nil).Times(1)

	var reply share.CreateReply
	err := s.Create(&arg, &reply)
	assert.Nil(t, err, "wrong Create")
	assert.Equal(t, info.TxId, reply.TxId, "wrong tx ID")
	assert.Equal(t, info.Id, reply.PayId, "wrong pay ID")
	assert.Equal(t, info.IssueTxId, reply.ShareId, "wrong issue tx ID")
	assert.Equal(t, *info.Payments[0][0], *reply.Payments[info.Payments[0][0].Currency.String()][0], "wrong payments")
}

func TestShareBalance(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	mode.Initialise(chain.Testing)
	defer mode.Finalise()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	r := mocks.NewMockReservoir(ctl)

	s := share.New(
		logger.New(fixtures.LogCategory),
		func(_ mode.Mode) bool { return true },
		r,
	)

	acc := account.Account{
		AccountInterface: &account.ED25519Account{
			Test:      true,
			PublicKey: fixtures.IssuerPrivateKey,
		},
	}

	arg := share.BalanceArguments{
		Owner:   &acc,
		ShareId: merkle.Digest{5, 3, 1},
		Count:   1000,
	}

	info := reservoir.BalanceInfo{
		ShareId:   arg.ShareId,
		Confirmed: uint64(5),
		Spend:     100,
		Available: 900,
	}

	r.EXPECT().ShareBalance(arg.Owner, arg.ShareId, arg.Count).Return([]reservoir.BalanceInfo{info}, nil).Times(1)

	var reply share.BalanceReply
	err := s.Balance(&arg, &reply)
	assert.Nil(t, err, "wrong Balance")
	assert.Equal(t, 1, len(reply.Balances), "wrong balance count")
	assert.Equal(t, info, reply.Balances[0], "wrong balance")
}

func TestShareGrant(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	mode.Initialise(chain.Testing)
	defer mode.Finalise()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	r := mocks.NewMockReservoir(ctl)

	s := share.New(
		logger.New(fixtures.LogCategory),
		func(_ mode.Mode) bool { return true },
		r,
	)

	acc1 := account.Account{
		AccountInterface: &account.ED25519Account{
			Test:      true,
			PublicKey: fixtures.IssuerPublicKey,
		},
	}

	acc2 := account.Account{
		AccountInterface: &account.ED25519Account{
			Test:      true,
			PublicKey: fixtures.ReceiverPublicKey,
		},
	}

	arg := transactionrecord.ShareGrant{
		ShareId:          merkle.Digest{},
		Quantity:         100,
		Owner:            &acc1,
		Recipient:        &acc2,
		BeforeBlock:      500,
		Signature:        nil,
		Countersignature: nil,
	}
	packed, _ := arg.Pack(&acc1)
	arg.Signature = ed25519.Sign(fixtures.IssuerPrivateKey, packed)
	packed, _ = arg.Pack(&acc2)
	arg.Countersignature = ed25519.Sign(fixtures.ReceiverPrivateKey, packed)

	info := reservoir.GrantInfo{
		Remaining: 1234,
		Id:        pay.PayId{1, 2, 3, 4},
		TxId:      merkle.Digest{5, 6, 7, 8},
		Packed:    nil,
		Payments: []transactionrecord.PaymentAlternative{
			[]*transactionrecord.Payment{
				{
					Currency: currency.Litecoin,
					Address:  fixtures.LitecoinAddress,
					Amount:   299,
				},
			},
		},
	}

	r.EXPECT().StoreGrant(&arg).Return(&info, false, nil).Times(1)

	var reply share.GrantReply
	err := s.Grant(&arg, &reply)
	assert.Nil(t, err, "wrong Grant")
	assert.Equal(t, info.TxId, reply.TxId, "wrong tx ID")
	assert.Equal(t, info.Id, reply.PayId, "wrong payment ID")
	assert.Equal(t, info.Remaining, reply.Remaining, "wrong remaining")
	assert.Equal(t, *info.Payments[0][0], *reply.Payments[info.Payments[0][0].Currency.String()][0], "wrong payments")
}

func TestShareSwap(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	mode.Initialise(chain.Testing)
	defer mode.Finalise()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	bus := messagebus.Bus.P2P.Chan()
	defer messagebus.Bus.P2P.Release()

	r := mocks.NewMockReservoir(ctl)

	s := share.New(
		logger.New(fixtures.LogCategory),
		func(_ mode.Mode) bool { return true },
		r,
	)

	acc1 := account.Account{
		AccountInterface: &account.ED25519Account{
			Test:      true,
			PublicKey: fixtures.IssuerPublicKey,
		},
	}

	acc2 := account.Account{
		AccountInterface: &account.ED25519Account{
			Test:      true,
			PublicKey: fixtures.ReceiverPublicKey,
		},
	}

	arg := transactionrecord.ShareSwap{
		ShareIdOne:       merkle.Digest{1, 2, 3, 4},
		QuantityOne:      5,
		OwnerOne:         &acc1,
		ShareIdTwo:       merkle.Digest{4, 3, 2, 1},
		QuantityTwo:      50,
		OwnerTwo:         &acc2,
		BeforeBlock:      200,
		Signature:        nil,
		Countersignature: nil,
	}
	packed, _ := arg.Pack(&acc1)
	arg.Signature = ed25519.Sign(fixtures.IssuerPrivateKey, packed)
	packed, _ = arg.Pack(&acc2)
	arg.Countersignature = ed25519.Sign(fixtures.ReceiverPrivateKey, packed)

	info := reservoir.SwapInfo{
		RemainingOne: 20,
		RemainingTwo: 30,
		Id:           pay.PayId{9, 9, 9, 9},
		TxId:         merkle.Digest{8, 8, 8, 8},
		Packed:       nil,
		Payments: []transactionrecord.PaymentAlternative{
			[]*transactionrecord.Payment{
				{
					Currency: currency.Litecoin,
					Address:  fixtures.LitecoinAddress,
					Amount:   299,
				},
			},
		},
	}

	r.EXPECT().StoreSwap(&arg).Return(&info, false, nil).Times(1)

	var reply share.SwapReply
	err := s.Swap(&arg, &reply)
	assert.Nil(t, err, "wrong Swap")
	assert.Equal(t, info.Id, reply.PayId, "wrong pay ID")
	assert.Equal(t, info.TxId, reply.TxId, "wrong tx ID")
	assert.Equal(t, info.RemainingOne, reply.RemainingOne, "wrong remaining one")
	assert.Equal(t, info.RemainingTwo, reply.RemainingTwo, "wrong remaining two")
	assert.Equal(t, *info.Payments[0][0], *reply.Payments[info.Payments[0][0].Currency.String()][0], "wrong payments")

	received := <-bus
	assert.Equal(t, "transfer", received.Command, "wrong message")
}
