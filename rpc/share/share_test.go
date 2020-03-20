// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package share_test

import (
	"crypto/ed25519"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/chain"
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/bitmarkd/rpc/fixtures"
	"github.com/bitmark-inc/bitmarkd/rpc/mocks"
	"github.com/bitmark-inc/bitmarkd/rpc/share"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
)

func TestShareCreate(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	bus := messagebus.Bus.Broadcast.Chan(5)
	defer messagebus.Bus.Broadcast.Release()

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
		Packed:    []byte{9},
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

	received := <-bus
	assert.Equal(t, "transfer", received.Command, "wrong command")
}

func TestShareCreateWhenNotNormal(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	r := mocks.NewMockReservoir(ctl)

	s := share.New(
		logger.New(fixtures.LogCategory),
		func(_ mode.Mode) bool { return false },
		r,
	)

	var reply share.CreateReply
	err := s.Create(&transactionrecord.BitmarkShare{}, &reply)
	assert.NotNil(t, err, "wrong Create")
	assert.Equal(t, fault.NotAvailableDuringSynchronise, err, "wrong error")
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

func TestShareBalanceWhenInvalidArgument(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	s := share.New(
		logger.New(fixtures.LogCategory),
		func(_ mode.Mode) bool { return true },
		nil,
	)

	var reply share.BalanceReply
	err := s.Balance(&share.BalanceArguments{}, &reply)
	assert.NotNil(t, err, "wrong Balance")
	assert.Equal(t, fault.InvalidItem, err, "wrong error")
}

func TestShareBalanceWhenInvalidOwner(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	s := share.New(
		logger.New(fixtures.LogCategory),
		func(_ mode.Mode) bool { return true },
		nil,
	)

	arg := share.BalanceArguments{
		Owner:   nil,
		ShareId: merkle.Digest{1, 2, 3, 4},
		Count:   0,
	}

	var reply share.BalanceReply
	err := s.Balance(&arg, &reply)
	assert.NotNil(t, err, "wrong Balance")
	assert.Equal(t, fault.InvalidItem, err, "wrong error")
}

func TestShareBalanceWhenNotNormal(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	s := share.New(
		logger.New(fixtures.LogCategory),
		func(_ mode.Mode) bool { return false },
		nil,
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

	var reply share.BalanceReply
	err := s.Balance(&arg, &reply)
	assert.NotNil(t, err, "wrong Balance")
	assert.Equal(t, fault.NotAvailableDuringSynchronise, err, "wrong error")
}

func TestShareBalanceWhenChainNotMatch(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	mode.Initialise(chain.Bitmark)
	defer mode.Finalise()

	s := share.New(
		logger.New(fixtures.LogCategory),
		func(_ mode.Mode) bool { return true },
		nil,
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

	var reply share.BalanceReply
	err := s.Balance(&arg, &reply)
	assert.NotNil(t, err, "wrong Balance")
	assert.Equal(t, fault.WrongNetworkForPublicKey, err, "wrong error")
}

func TestShareBalanceWhenSmallCount(t *testing.T) {
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

	arg := share.BalanceArguments{
		Owner:   &account.Account{},
		ShareId: merkle.Digest{5, 3, 1},
		Count:   0,
	}

	var reply share.BalanceReply
	err := s.Balance(&arg, &reply)
	assert.NotNil(t, err, "wrong Balance")
	assert.Equal(t, fault.InvalidCount, err, "wrong error")
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
		Packed:    []byte{3},
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

	messagebus.Bus.Broadcast.Release()
	bus := messagebus.Bus.Broadcast.Chan(5)
	defer messagebus.Bus.Broadcast.Release()

	var reply share.GrantReply
	err := s.Grant(&arg, &reply)
	received := <-bus
	assert.Equal(t, "transfer", received.Command, "wrong command")
	assert.Nil(t, err, "wrong Grant")
	assert.Equal(t, info.TxId, reply.TxId, "wrong tx ID")
	assert.Equal(t, info.Id, reply.PayId, "wrong payment ID")
	assert.Equal(t, info.Remaining, reply.Remaining, "wrong remaining")
	assert.Equal(t, *info.Payments[0][0], *reply.Payments[info.Payments[0][0].Currency.String()][0], "wrong payments")

}

func TestShareGrantWhenEmptyArguments(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	s := share.New(
		logger.New(fixtures.LogCategory),
		func(_ mode.Mode) bool { return true },
		nil,
	)

	var reply share.GrantReply
	err := s.Grant(&transactionrecord.ShareGrant{}, &reply)
	assert.NotNil(t, err, "wrong Grant")
	assert.Equal(t, fault.InvalidItem, err, "wrong error")
}

func TestShareGrantWhenEmptyArgumentsOwner(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	s := share.New(
		logger.New(fixtures.LogCategory),
		func(_ mode.Mode) bool { return true },
		nil,
	)

	arg := transactionrecord.ShareGrant{
		ShareId:          merkle.Digest{},
		Quantity:         100,
		Owner:            nil,
		Recipient:        nil,
		BeforeBlock:      500,
		Signature:        nil,
		Countersignature: nil,
	}

	var reply share.GrantReply
	err := s.Grant(&arg, &reply)
	assert.NotNil(t, err, "wrong Grant")
	assert.Equal(t, fault.InvalidItem, err, "wrong error")
}

func TestShareGrantWhenEmptyArgumentsRecipient(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	s := share.New(
		logger.New(fixtures.LogCategory),
		func(_ mode.Mode) bool { return true },
		nil,
	)

	acc1 := account.Account{
		AccountInterface: &account.ED25519Account{
			Test:      true,
			PublicKey: fixtures.IssuerPublicKey,
		},
	}

	arg := transactionrecord.ShareGrant{
		ShareId:          merkle.Digest{},
		Quantity:         100,
		Owner:            &acc1,
		Recipient:        nil,
		BeforeBlock:      500,
		Signature:        nil,
		Countersignature: nil,
	}
	packed, _ := arg.Pack(&acc1)
	arg.Signature = ed25519.Sign(fixtures.IssuerPrivateKey, packed)

	var reply share.GrantReply
	err := s.Grant(&arg, &reply)
	assert.NotNil(t, err, "wrong Grant")
	assert.Equal(t, fault.InvalidItem, err, "wrong error")
}

func TestShareGrantNotEnoughQuantity(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	s := share.New(
		logger.New(fixtures.LogCategory),
		func(_ mode.Mode) bool { return false },
		nil,
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
		Quantity:         0,
		Owner:            &acc1,
		Recipient:        &acc2,
		BeforeBlock:      500,
		Signature:        nil,
		Countersignature: nil,
	}

	var reply share.GrantReply
	err := s.Grant(&arg, &reply)
	assert.NotNil(t, err, "wrong Grant")
	assert.Equal(t, fault.ShareQuantityTooSmall, err, "wrong error")
}

func TestShareGrantNotNormal(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	s := share.New(
		logger.New(fixtures.LogCategory),
		func(_ mode.Mode) bool { return false },
		nil,
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

	var reply share.GrantReply
	err := s.Grant(&arg, &reply)
	assert.NotNil(t, err, "wrong Grant")
	assert.Equal(t, fault.NotAvailableDuringSynchronise, err, "wrong error")
}

func TestShareGrantOwnerChainDifferent(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	mode.Initialise(chain.Bitmark)
	defer mode.Finalise()

	s := share.New(
		logger.New(fixtures.LogCategory),
		func(_ mode.Mode) bool { return true },
		nil,
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

	var reply share.GrantReply
	err := s.Grant(&arg, &reply)
	assert.NotNil(t, err, "wrong Grant")
	assert.Equal(t, fault.WrongNetworkForPublicKey, err, "wrong error")
}

func TestShareGrantRecipientChainDifferent(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	mode.Initialise(chain.Bitmark)
	defer mode.Finalise()

	s := share.New(
		logger.New(fixtures.LogCategory),
		func(_ mode.Mode) bool { return true },
		nil,
	)

	acc1 := account.Account{
		AccountInterface: &account.ED25519Account{
			Test:      false,
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

	var reply share.GrantReply
	err := s.Grant(&arg, &reply)
	assert.NotNil(t, err, "wrong Grant")
	assert.Equal(t, fault.WrongNetworkForPublicKey, err, "wrong error")
}

func TestShareGrantWhenStoreError(t *testing.T) {
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

	r.EXPECT().StoreGrant(&arg).Return(nil, false, fmt.Errorf("fake")).Times(1)

	var reply share.GrantReply
	err := s.Grant(&arg, &reply)
	assert.NotNil(t, err, "wrong Balance")
	assert.Equal(t, "fake", err.Error(), "wrong error")
}

func TestShareSwap(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	mode.Initialise(chain.Testing)
	defer mode.Finalise()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	bus := messagebus.Bus.Broadcast.Chan(5)
	defer messagebus.Bus.Broadcast.Release()

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
		Packed:       []byte{7},
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

func TestShareSwapWhenEmptyArguments(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	s := share.New(
		logger.New(fixtures.LogCategory),
		func(_ mode.Mode) bool { return true },
		nil,
	)

	var reply share.SwapReply
	err := s.Swap(&transactionrecord.ShareSwap{}, &reply)
	assert.NotNil(t, err, "wrong Swap")
	assert.Equal(t, fault.InvalidItem, err, "wrong error")
}

func TestShareSwapWhenEmptyOwnerOne(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	s := share.New(
		logger.New(fixtures.LogCategory),
		func(_ mode.Mode) bool { return true },
		nil,
	)

	acc2 := account.Account{
		AccountInterface: &account.ED25519Account{
			Test:      true,
			PublicKey: fixtures.ReceiverPublicKey,
		},
	}

	arg := transactionrecord.ShareSwap{
		ShareIdOne:       merkle.Digest{1, 2, 3, 4},
		QuantityOne:      5,
		OwnerOne:         nil,
		ShareIdTwo:       merkle.Digest{4, 3, 2, 1},
		QuantityTwo:      50,
		OwnerTwo:         &acc2,
		BeforeBlock:      200,
		Signature:        nil,
		Countersignature: nil,
	}

	var reply share.SwapReply
	err := s.Swap(&arg, &reply)
	assert.NotNil(t, err, "wrong Swap")
	assert.Equal(t, fault.InvalidItem, err, "wrong error")
}

func TestShareSwapWhenEmptyOwnerTwo(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	s := share.New(
		logger.New(fixtures.LogCategory),
		func(_ mode.Mode) bool { return true },
		nil,
	)

	acc1 := account.Account{
		AccountInterface: &account.ED25519Account{
			Test:      true,
			PublicKey: fixtures.IssuerPublicKey,
		},
	}

	arg := transactionrecord.ShareSwap{
		ShareIdOne:       merkle.Digest{1, 2, 3, 4},
		QuantityOne:      5,
		OwnerOne:         &acc1,
		ShareIdTwo:       merkle.Digest{4, 3, 2, 1},
		QuantityTwo:      50,
		OwnerTwo:         nil,
		BeforeBlock:      200,
		Signature:        nil,
		Countersignature: nil,
	}

	var reply share.SwapReply
	err := s.Swap(&arg, &reply)
	assert.NotNil(t, err, "wrong Swap")
	assert.Equal(t, fault.InvalidItem, err, "wrong error")
}

func TestShareSwapWhenQuantityOneNotEnough(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	s := share.New(
		logger.New(fixtures.LogCategory),
		func(_ mode.Mode) bool { return true },
		nil,
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
		QuantityOne:      0,
		OwnerOne:         &acc1,
		ShareIdTwo:       merkle.Digest{4, 3, 2, 1},
		QuantityTwo:      50,
		OwnerTwo:         &acc2,
		BeforeBlock:      200,
		Signature:        nil,
		Countersignature: nil,
	}

	var reply share.SwapReply
	err := s.Swap(&arg, &reply)
	assert.NotNil(t, err, "wrong Swap")
	assert.Equal(t, fault.ShareQuantityTooSmall, err, "wrong error")
}

func TestShareSwapWhenQuantityTowNotEnough(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	s := share.New(
		logger.New(fixtures.LogCategory),
		func(_ mode.Mode) bool { return true },
		nil,
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
		QuantityOne:      500,
		OwnerOne:         &acc1,
		ShareIdTwo:       merkle.Digest{4, 3, 2, 1},
		QuantityTwo:      0,
		OwnerTwo:         &acc2,
		BeforeBlock:      200,
		Signature:        nil,
		Countersignature: nil,
	}

	var reply share.SwapReply
	err := s.Swap(&arg, &reply)
	assert.NotNil(t, err, "wrong Swap")
	assert.Equal(t, fault.ShareQuantityTooSmall, err, "wrong error")
}

func TestShareSwapWhenNotNormal(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	s := share.New(
		logger.New(fixtures.LogCategory),
		func(_ mode.Mode) bool { return false },
		nil,
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

	var reply share.SwapReply
	err := s.Swap(&arg, &reply)
	assert.NotNil(t, err, "wrong Swap")
	assert.Equal(t, fault.NotAvailableDuringSynchronise, err, "wrong error")
}

func TestShareSwapWhenOwnerOneChainDifferent(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	mode.Initialise(chain.Bitmark)
	defer mode.Finalise()

	s := share.New(
		logger.New(fixtures.LogCategory),
		func(_ mode.Mode) bool { return true },
		nil,
	)

	acc1 := account.Account{
		AccountInterface: &account.ED25519Account{
			Test:      false,
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

	var reply share.SwapReply
	err := s.Swap(&arg, &reply)
	assert.NotNil(t, err, "wrong Swap")
	assert.Equal(t, fault.WrongNetworkForPublicKey, err, "wrong error")
}

func TestShareSwapWhenOwnerTowChainDifferent(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	mode.Initialise(chain.Bitmark)
	defer mode.Finalise()

	s := share.New(
		logger.New(fixtures.LogCategory),
		func(_ mode.Mode) bool { return true },
		nil,
	)

	acc1 := account.Account{
		AccountInterface: &account.ED25519Account{
			Test:      true,
			PublicKey: fixtures.IssuerPublicKey,
		},
	}

	acc2 := account.Account{
		AccountInterface: &account.ED25519Account{
			Test:      false,
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

	var reply share.SwapReply
	err := s.Swap(&arg, &reply)
	assert.NotNil(t, err, "wrong Swap")
	assert.Equal(t, fault.WrongNetworkForPublicKey, err, "wrong error")
}

func TestShareSwapWhenStoreSwapError(t *testing.T) {
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

	r.EXPECT().StoreSwap(gomock.Any()).Return(nil, false, fmt.Errorf("fake")).Times(1)

	var reply share.SwapReply
	err := s.Swap(&arg, &reply)
	assert.NotNil(t, err, "wrong Swap")
	assert.Equal(t, "fake", err.Error(), "wrong error")
}
