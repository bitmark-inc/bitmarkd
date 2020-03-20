// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package bitmark_test

import (
	"crypto/ed25519"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/chain"
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/bitmarkd/rpc/bitmark"
	"github.com/bitmark-inc/bitmarkd/rpc/fixtures"
	"github.com/bitmark-inc/bitmarkd/rpc/mocks"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
)

func TestBitmarkTransfer(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	mode.Initialise(chain.Testing)
	defer mode.Finalise()

	bus := messagebus.Bus.Broadcast.Chan(5)
	defer messagebus.Bus.Broadcast.Release()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	owner := account.Account{
		AccountInterface: &account.ED25519Account{
			Test:      true,
			PublicKey: fixtures.IssuerPublicKey,
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

	r := mocks.NewMockReservoir(ctl)
	r.EXPECT().StoreTransfer(&unratitifed).Return(&info, false, nil).Times(1)

	b := bitmark.New(
		logger.New(fixtures.LogCategory),
		reservoir.Handles{},
		func(_ mode.Mode) bool { return true },
		func() bool { return true },
		r,
	)

	var reply bitmark.TransferReply
	err := b.Transfer(&transfer, &reply)
	assert.Nil(t, err, "wrong transfer")
	assert.Equal(t, info.Id, reply.PayId, "wrong payID")
	assert.Equal(t, info.TxId, reply.TxId, "wrong txID")
	assert.Equal(t, 1, len(reply.Payments), "wrong payment count")
	assert.Equal(t, fixtures.LitecoinAddress, reply.Payments[currency.Litecoin.String()][0].Address, "wrong litecoin payment address")

	received := <-bus
	assert.Equal(t, "transfer", received.Command, "wrong message")
}

func TestBitmarkProvenanceWhenBitmarkIssuance(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	mode.Initialise(chain.Testing)
	defer mode.Finalise()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	r := mocks.NewMockReservoir(ctl)
	poolT := mocks.NewMockHandle(ctl)
	poolA := mocks.NewMockHandle(ctl)
	poolO := mocks.NewMockHandle(ctl)

	b := bitmark.New(
		logger.New(fixtures.LogCategory),
		reservoir.Handles{
			Assets:       poolA,
			Transactions: poolT,
			OwnerTxIndex: poolO,
		},
		func(_ mode.Mode) bool { return true },
		func() bool { return true },
		r,
	)

	txID := merkle.Digest{1, 2, 3, 4}

	arg := bitmark.ProvenanceArguments{
		TxId:  txID,
		Count: 2,
	}

	acc := account.Account{
		AccountInterface: &account.ED25519Account{
			Test:      true,
			PublicKey: fixtures.IssuerPublicKey,
		},
	}

	tr1 := transactionrecord.BitmarkIssue{
		AssetId:   transactionrecord.AssetIdentifier{},
		Owner:     &acc,
		Nonce:     1,
		Signature: nil,
	}
	packed1, _ := tr1.Pack(&acc)
	tr1.Signature = ed25519.Sign(fixtures.IssuerPrivateKey, packed1)
	packed1, _ = tr1.Pack(&acc)

	ass := transactionrecord.AssetData{
		Name:        "test",
		Fingerprint: "fin",
		Metadata:    "owner\x00me",
		Registrant:  &acc,
		Signature:   nil,
	}
	packed2, _ := ass.Pack(&acc)
	ass.Signature = ed25519.Sign(fixtures.IssuerPrivateKey, packed2)
	packed2, _ = ass.Pack(&acc)

	poolT.EXPECT().GetNB(txID[:]).Return(uint64(1), packed1).Times(1)
	poolO.EXPECT().Has(gomock.Any()).Return(true).Times(1)
	poolA.EXPECT().GetNB(gomock.Any()).Return(uint64(1), packed2).Times(1)

	var reply bitmark.ProvenanceReply
	err := b.Provenance(&arg, &reply)
	assert.Nil(t, err, "wrong Provenance")
	assert.Equal(t, 2, len(reply.Data), "wrong reply count")
	assert.Equal(t, "BitmarkIssue", reply.Data[0].Record, "wrong record name")
	assert.True(t, reply.Data[0].IsOwner, "wrong is owner")
	assert.Equal(t, txID, reply.Data[0].TxId, "wrong tx ID")

	assert.Equal(t, "AssetData", reply.Data[1].Record, "wrong record name")
	d := reply.Data[1].Data.(*transactionrecord.AssetData)
	assert.Equal(t, ass.Name, d.Name, "wrong asset name")
	assert.Equal(t, ass.Fingerprint, d.Fingerprint, "wrong asset fingerprint")
	assert.Equal(t, ass.Metadata, d.Metadata, "wrong meta data")
	assert.Equal(t, &acc, d.Registrant, "wrong registrant")
}

func TestBitmarkProvenanceWhenOldBaseData(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	mode.Initialise(chain.Testing)
	defer mode.Finalise()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	r := mocks.NewMockReservoir(ctl)
	poolT := mocks.NewMockHandle(ctl)
	poolA := mocks.NewMockHandle(ctl)
	poolO := mocks.NewMockHandle(ctl)

	b := bitmark.New(
		logger.New(fixtures.LogCategory),
		reservoir.Handles{
			Assets:       poolA,
			Transactions: poolT,
			OwnerTxIndex: poolO,
		},
		func(_ mode.Mode) bool { return true },
		func() bool { return true },
		r,
	)

	txID := merkle.Digest{1, 2, 3, 4}

	arg := bitmark.ProvenanceArguments{
		TxId:  txID,
		Count: 2,
	}

	acc := account.Account{
		AccountInterface: &account.ED25519Account{
			Test:      true,
			PublicKey: fixtures.IssuerPublicKey,
		},
	}

	tr1 := transactionrecord.OldBaseData{
		Currency:       currency.Litecoin,
		PaymentAddress: fixtures.LitecoinAddress,
		Owner:          &acc,
		Nonce:          1,
		Signature:      nil,
	}
	packed1, _ := tr1.Pack(&acc)
	tr1.Signature = ed25519.Sign(fixtures.IssuerPrivateKey, packed1)
	packed1, _ = tr1.Pack(&acc)

	poolT.EXPECT().GetNB(txID[:]).Return(uint64(1), packed1).Times(1)
	poolO.EXPECT().Has(gomock.Any()).Return(true).Times(1)

	var reply bitmark.ProvenanceReply
	err := b.Provenance(&arg, &reply)
	assert.Nil(t, err, "wrong Provenance")
	assert.Equal(t, 1, len(reply.Data), "wrong reply count")
	assert.Equal(t, "BaseData", reply.Data[0].Record, "wrong record name")
	assert.True(t, reply.Data[0].IsOwner, "wrong is owner")
	assert.Equal(t, txID, reply.Data[0].TxId, "wrong tx ID")
	assert.Equal(t, &tr1, reply.Data[0].Data, "wrong data")
}

func TestBitmarkProvenanceWhenBlockFoundation(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	mode.Initialise(chain.Testing)
	defer mode.Finalise()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	r := mocks.NewMockReservoir(ctl)
	poolT := mocks.NewMockHandle(ctl)
	poolA := mocks.NewMockHandle(ctl)
	poolO := mocks.NewMockHandle(ctl)

	b := bitmark.New(
		logger.New(fixtures.LogCategory),
		reservoir.Handles{
			Assets:       poolA,
			Transactions: poolT,
			OwnerTxIndex: poolO,
		},
		func(_ mode.Mode) bool { return true },
		func() bool { return true },
		r,
	)

	txID := merkle.Digest{1, 2, 3, 4}

	arg := bitmark.ProvenanceArguments{
		TxId:  txID,
		Count: 2,
	}

	acc := account.Account{
		AccountInterface: &account.ED25519Account{
			Test:      true,
			PublicKey: fixtures.IssuerPublicKey,
		},
	}

	tr1 := transactionrecord.BlockFoundation{
		Version: uint64(1),
		Payments: map[currency.Currency]string{
			currency.Bitcoin:  fixtures.BitcoinAddress,
			currency.Litecoin: fixtures.LitecoinAddress,
		},
		Owner:     &acc,
		Nonce:     1,
		Signature: nil,
	}
	packed1, _ := tr1.Pack(&acc)
	tr1.Signature = ed25519.Sign(fixtures.IssuerPrivateKey, packed1)
	packed1, _ = tr1.Pack(&acc)

	poolT.EXPECT().GetNB(txID[:]).Return(uint64(1), packed1).Times(1)
	poolO.EXPECT().Has(gomock.Any()).Return(false).Times(1)

	var reply bitmark.ProvenanceReply
	err := b.Provenance(&arg, &reply)
	assert.Nil(t, err, "wrong Provenance")
	assert.Equal(t, 1, len(reply.Data), "wrong reply count")
	assert.Equal(t, "BlockFoundation", reply.Data[0].Record, "wrong record name")
	assert.False(t, reply.Data[0].IsOwner, "wrong is owner")
	assert.Equal(t, txID, reply.Data[0].TxId, "wrong tx ID")
	assert.Equal(t, &tr1, reply.Data[0].Data, "wrong data")
}

func TestBitmarkProvenanceWhenTransferUnratified(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	mode.Initialise(chain.Testing)
	defer mode.Finalise()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	r := mocks.NewMockReservoir(ctl)
	poolT := mocks.NewMockHandle(ctl)
	poolA := mocks.NewMockHandle(ctl)
	poolO := mocks.NewMockHandle(ctl)

	b := bitmark.New(
		logger.New(fixtures.LogCategory),
		reservoir.Handles{
			Assets:       poolA,
			Transactions: poolT,
			OwnerTxIndex: poolO,
		},
		func(_ mode.Mode) bool { return true },
		func() bool { return true },
		r,
	)

	txID := merkle.Digest{1, 2, 3, 4}

	arg := bitmark.ProvenanceArguments{
		TxId:  txID,
		Count: 2,
	}

	acc := account.Account{
		AccountInterface: &account.ED25519Account{
			Test:      true,
			PublicKey: fixtures.IssuerPublicKey,
		},
	}

	tr1 := transactionrecord.BitmarkTransferUnratified{
		Link:      merkle.Digest{},
		Escrow:    nil,
		Owner:     &acc,
		Signature: nil,
	}
	packed1, _ := tr1.Pack(&acc)
	tr1.Signature = ed25519.Sign(fixtures.IssuerPrivateKey, packed1)
	packed1, _ = tr1.Pack(&acc)

	poolT.EXPECT().GetNB(txID[:]).Return(uint64(1), packed1).Times(1)
	poolT.EXPECT().GetNB(merkle.Digest{}.Bytes()).Return(uint64(0), nil).Times(1)
	poolO.EXPECT().Has(gomock.Any()).Return(true).Times(1)

	var reply bitmark.ProvenanceReply
	err := b.Provenance(&arg, &reply)
	assert.Nil(t, err, "wrong Provenance")
	assert.Equal(t, 1, len(reply.Data), "wrong reply count")
	assert.Equal(t, "BitmarkTransferUnratified", reply.Data[0].Record, "wrong record name")
	assert.True(t, reply.Data[0].IsOwner, "wrong is owner")
	assert.Equal(t, txID, reply.Data[0].TxId, "wrong tx ID")
	assert.Equal(t, &tr1, reply.Data[0].Data, "wrong data")
}

func TestBitmarkProvenanceWhenBitmarkShare(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	mode.Initialise(chain.Testing)
	defer mode.Finalise()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	r := mocks.NewMockReservoir(ctl)
	poolT := mocks.NewMockHandle(ctl)
	poolA := mocks.NewMockHandle(ctl)
	poolO := mocks.NewMockHandle(ctl)

	b := bitmark.New(
		logger.New(fixtures.LogCategory),
		reservoir.Handles{
			Assets:       poolA,
			Transactions: poolT,
			OwnerTxIndex: poolO,
		},
		func(_ mode.Mode) bool { return true },
		func() bool { return true },
		r,
	)

	txID := merkle.Digest{1, 2, 3, 4}

	arg := bitmark.ProvenanceArguments{
		TxId:  txID,
		Count: 2,
	}

	acc := account.Account{
		AccountInterface: &account.ED25519Account{
			Test:      true,
			PublicKey: fixtures.IssuerPublicKey,
		},
	}

	tr1 := transactionrecord.BitmarkShare{
		Link:      txID,
		Quantity:  5,
		Signature: nil,
	}
	packed1, _ := tr1.Pack(&acc)
	tr1.Signature = ed25519.Sign(fixtures.IssuerPrivateKey, packed1)
	packed1, _ = tr1.Pack(&acc)

	poolT.EXPECT().GetNB(txID[:]).Return(uint64(1), packed1).Times(1)
	poolT.EXPECT().GetNB(txID[:]).Return(uint64(0), nil).Times(1)

	var reply bitmark.ProvenanceReply
	err := b.Provenance(&arg, &reply)
	assert.Nil(t, err, "wrong Provenance")
	assert.Equal(t, 1, len(reply.Data), "wrong reply count")
	assert.Equal(t, "ShareBalance", reply.Data[0].Record, "wrong record name")
	assert.True(t, reply.Data[0].IsOwner, "wrong is owner")
	assert.Equal(t, txID, reply.Data[0].TxId, "wrong tx ID")
	assert.Equal(t, &tr1, reply.Data[0].Data, "wrong data")
}
