// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc_test

import (
	"crypto/ed25519"
	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/chain"
	"github.com/bitmark-inc/bitmarkd/currency"
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

func TestBitmarkProvenanceWhenBitmarkIssuance(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()

	mode.Initialise(chain.Testing)
	defer mode.Finalise()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	r := mocks.NewMockReservoir(ctl)
	poolT := mocks.NewMockHandle(ctl)
	poolA := mocks.NewMockHandle(ctl)
	poolO := mocks.NewMockHandle(ctl)

	b := rpc.Bitmark{
		Log:              logger.New(logCategory),
		Limiter:          rate.NewLimiter(100, 100),
		IsNormalMode:     func(_ mode.Mode) bool { return true },
		IsTestingChain:   func() bool { return true },
		Rsvr:             r,
		PoolTransactions: poolT,
		PoolAssets:       poolA,
		PoolOwnerTxIndex: poolO,
	}

	txID := merkle.Digest{1, 2, 3, 4}

	arg := rpc.ProvenanceArguments{
		TxId:  txID,
		Count: 2,
	}

	acc := account.Account{
		AccountInterface: &account.ED25519Account{
			Test:      true,
			PublicKey: issuerPublicKey,
		},
	}

	tr1 := transactionrecord.BitmarkIssue{
		AssetId:   transactionrecord.AssetIdentifier{},
		Owner:     &acc,
		Nonce:     1,
		Signature: nil,
	}
	packed1, _ := tr1.Pack(&acc)
	tr1.Signature = ed25519.Sign(issuerPrivateKey, packed1)
	packed1, _ = tr1.Pack(&acc)

	ass := transactionrecord.AssetData{
		Name:        "test",
		Fingerprint: "fin",
		Metadata:    "owner\x00me",
		Registrant:  &acc,
		Signature:   nil,
	}
	packed2, _ := ass.Pack(&acc)
	ass.Signature = ed25519.Sign(issuerPrivateKey, packed2)
	packed2, _ = ass.Pack(&acc)

	poolT.EXPECT().GetNB(txID[:]).Return(uint64(1), packed1).Times(1)
	poolO.EXPECT().Has(gomock.Any()).Return(true).Times(1)
	poolA.EXPECT().GetNB(gomock.Any()).Return(uint64(1), packed2).Times(1)

	var reply rpc.ProvenanceReply
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
	setupTestLogger()
	defer teardownTestLogger()

	mode.Initialise(chain.Testing)
	defer mode.Finalise()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	r := mocks.NewMockReservoir(ctl)
	poolT := mocks.NewMockHandle(ctl)
	poolA := mocks.NewMockHandle(ctl)
	poolO := mocks.NewMockHandle(ctl)

	b := rpc.Bitmark{
		Log:              logger.New(logCategory),
		Limiter:          rate.NewLimiter(100, 100),
		IsNormalMode:     func(_ mode.Mode) bool { return true },
		IsTestingChain:   func() bool { return true },
		Rsvr:             r,
		PoolTransactions: poolT,
		PoolAssets:       poolA,
		PoolOwnerTxIndex: poolO,
	}

	txID := merkle.Digest{1, 2, 3, 4}

	arg := rpc.ProvenanceArguments{
		TxId:  txID,
		Count: 2,
	}

	acc := account.Account{
		AccountInterface: &account.ED25519Account{
			Test:      true,
			PublicKey: issuerPublicKey,
		},
	}

	tr1 := transactionrecord.OldBaseData{
		Currency:       currency.Litecoin,
		PaymentAddress: litecoinAddress,
		Owner:          &acc,
		Nonce:          1,
		Signature:      nil,
	}
	packed1, _ := tr1.Pack(&acc)
	tr1.Signature = ed25519.Sign(issuerPrivateKey, packed1)
	packed1, _ = tr1.Pack(&acc)

	poolT.EXPECT().GetNB(txID[:]).Return(uint64(1), packed1).Times(1)
	poolO.EXPECT().Has(gomock.Any()).Return(true).Times(1)

	var reply rpc.ProvenanceReply
	err := b.Provenance(&arg, &reply)
	assert.Nil(t, err, "wrong Provenance")
	assert.Equal(t, 1, len(reply.Data), "wrong reply count")
	assert.Equal(t, "BaseData", reply.Data[0].Record, "wrong record name")
	assert.True(t, reply.Data[0].IsOwner, "wrong is owner")
	assert.Equal(t, txID, reply.Data[0].TxId, "wrong tx ID")
	assert.Equal(t, &tr1, reply.Data[0].Data, "wrong data")
}

func TestBitmarkProvenanceWhenBlockFoundation(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()

	mode.Initialise(chain.Testing)
	defer mode.Finalise()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	r := mocks.NewMockReservoir(ctl)
	poolT := mocks.NewMockHandle(ctl)
	poolA := mocks.NewMockHandle(ctl)
	poolO := mocks.NewMockHandle(ctl)

	b := rpc.Bitmark{
		Log:              logger.New(logCategory),
		Limiter:          rate.NewLimiter(100, 100),
		IsNormalMode:     func(_ mode.Mode) bool { return true },
		IsTestingChain:   func() bool { return true },
		Rsvr:             r,
		PoolTransactions: poolT,
		PoolAssets:       poolA,
		PoolOwnerTxIndex: poolO,
	}

	txID := merkle.Digest{1, 2, 3, 4}

	arg := rpc.ProvenanceArguments{
		TxId:  txID,
		Count: 2,
	}

	acc := account.Account{
		AccountInterface: &account.ED25519Account{
			Test:      true,
			PublicKey: issuerPublicKey,
		},
	}

	tr1 := transactionrecord.BlockFoundation{
		Version: uint64(1),
		Payments: map[currency.Currency]string{
			currency.Bitcoin:  bitcoinAddress,
			currency.Litecoin: litecoinAddress,
		},
		Owner:     &acc,
		Nonce:     1,
		Signature: nil,
	}
	packed1, _ := tr1.Pack(&acc)
	tr1.Signature = ed25519.Sign(issuerPrivateKey, packed1)
	packed1, _ = tr1.Pack(&acc)

	poolT.EXPECT().GetNB(txID[:]).Return(uint64(1), packed1).Times(1)
	poolO.EXPECT().Has(gomock.Any()).Return(false).Times(1)

	var reply rpc.ProvenanceReply
	err := b.Provenance(&arg, &reply)
	assert.Nil(t, err, "wrong Provenance")
	assert.Equal(t, 1, len(reply.Data), "wrong reply count")
	assert.Equal(t, "BlockFoundation", reply.Data[0].Record, "wrong record name")
	assert.False(t, reply.Data[0].IsOwner, "wrong is owner")
	assert.Equal(t, txID, reply.Data[0].TxId, "wrong tx ID")
	assert.Equal(t, &tr1, reply.Data[0].Data, "wrong data")
}
