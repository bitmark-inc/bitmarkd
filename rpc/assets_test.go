// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc_test

import (
	"crypto/ed25519"
	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/rpc"
	"github.com/bitmark-inc/bitmarkd/rpc/mocks"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
	"testing"
)

func TestAssetsGet(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	p := mocks.NewMockHandle(ctl)

	a := rpc.Assets{
		Log:            logger.New(logCategory),
		Limiter:        rate.NewLimiter(200, 100),
		Pool:           p,
		IsNormalMode:   func(_ mode.Mode) bool { return true },
		IsTestingChain: func() bool { return true },
	}

	arg := rpc.AssetGetArguments{Fingerprints: []string{"fin1", "fin2"}}
	var reply rpc.AssetGetReply
	bin1 := transactionrecord.NewAssetIdentifier([]byte("fin1"))
	bin2 := transactionrecord.NewAssetIdentifier([]byte("fin2"))
	acc := &account.Account{
		AccountInterface: &account.ED25519Account{
			Test:      true,
			PublicKey: publicKey,
		},
	}
	ad := transactionrecord.AssetData{
		Name:        "test",
		Fingerprint: "123456789",
		Metadata:    "owner\x00test",
		Registrant:  acc,
	}
	packed, _ := ad.Pack(acc)
	signature := ed25519.Sign(privateKey, packed)
	ad.Signature = signature
	packed, _ = ad.Pack(acc)

	p.EXPECT().GetNB(bin1[:]).Return(uint64(1), packed).Times(1)
	p.EXPECT().GetNB(bin2[:]).Return(uint64(1), packed).Times(1)

	err := a.Get(&arg, &reply)
	assert.Nil(t, err, "wrong get")
	assert.Equal(t, 2, len(reply.Assets), "wrong asset count")

	assert.Equal(t, "AssetData", reply.Assets[0].Record, "wrong record")
	assert.True(t, reply.Assets[0].Confirmed, "wrong confirmed")

	d := reply.Assets[0].Data.(*transactionrecord.AssetData)
	assert.Equal(t, ad.Name, d.Name, "wrong asset name")
	assert.Equal(t, ad.Fingerprint, d.Fingerprint, "wrong asset fingerprint")
	assert.Equal(t, ad.Metadata, d.Metadata, "wrong asset metadata")
}

func TestAssetsGetWhenNotInNormal(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	p := mocks.NewMockHandle(ctl)

	a := rpc.Assets{
		Log:            logger.New(logCategory),
		Limiter:        rate.NewLimiter(200, 100),
		Pool:           p,
		IsNormalMode:   func(_ mode.Mode) bool { return false },
		IsTestingChain: func() bool { return true },
	}

	var reply rpc.AssetGetReply
	arg := rpc.AssetGetArguments{Fingerprints: []string{"fin1", "fin2"}}

	err := a.Get(&arg, &reply)
	assert.Equal(t, fault.NotAvailableDuringSynchronise, err, "wrong error")
}

func TestAssetsGetWhenNilAsset(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	p := mocks.NewMockHandle(ctl)

	a := rpc.Assets{
		Log:            logger.New(logCategory),
		Limiter:        rate.NewLimiter(200, 100),
		Pool:           p,
		IsNormalMode:   func(_ mode.Mode) bool { return true },
		IsTestingChain: func() bool { return true },
	}

	arg := rpc.AssetGetArguments{Fingerprints: []string{"fin1"}}
	var reply rpc.AssetGetReply
	bin1 := transactionrecord.NewAssetIdentifier([]byte("fin1"))
	acc := &account.Account{
		AccountInterface: &account.ED25519Account{
			Test:      true,
			PublicKey: publicKey,
		},
	}
	ad := transactionrecord.AssetData{
		Name:        "test",
		Fingerprint: "123456789",
		Metadata:    "owner\x00test",
		Registrant:  acc,
	}
	packed, _ := ad.Pack(acc)
	signature := ed25519.Sign(privateKey, packed)
	ad.Signature = signature
	packed, _ = ad.Pack(acc)

	p.EXPECT().GetNB(bin1[:]).Return(uint64(1), nil).Times(1)

	err := a.Get(&arg, &reply)
	assert.Nil(t, err, "wrong get")
	assert.Equal(t, 1, len(reply.Assets), "wrong asset count")

	assert.False(t, reply.Assets[0].Confirmed, "wrong confirmed")
}

func TestAssetsGetWhwnUnpackError(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	p := mocks.NewMockHandle(ctl)

	a := rpc.Assets{
		Log:            logger.New(logCategory),
		Limiter:        rate.NewLimiter(200, 100),
		Pool:           p,
		IsNormalMode:   func(_ mode.Mode) bool { return true },
		IsTestingChain: func() bool { return true },
	}

	arg := rpc.AssetGetArguments{Fingerprints: []string{"fin1", "fin2"}}
	var reply rpc.AssetGetReply
	bin1 := transactionrecord.NewAssetIdentifier([]byte("fin1"))
	bin2 := transactionrecord.NewAssetIdentifier([]byte("fin2"))
	acc := &account.Account{
		AccountInterface: &account.ED25519Account{
			Test:      true,
			PublicKey: publicKey,
		},
	}
	ad := transactionrecord.AssetData{
		Name:        "test",
		Fingerprint: "123456789",
		Metadata:    "owner\x00test",
		Registrant:  acc,
	}
	packed, _ := ad.Pack(acc)
	signature := ed25519.Sign(privateKey, packed)
	ad.Signature = signature
	packed, _ = ad.Pack(acc)

	p.EXPECT().GetNB(bin1[:]).Return(uint64(1), []byte{}).Times(1)
	p.EXPECT().GetNB(bin2[:]).Return(uint64(1), []byte{}).Times(1)

	err := a.Get(&arg, &reply)
	assert.Nil(t, err, "wrong get")
	assert.Equal(t, 2, len(reply.Assets), "wrong asset count")
	assert.Equal(t, "", reply.Assets[0].Record, "wrong record")
	assert.False(t, reply.Assets[0].Confirmed, "wrong confirmed")
}
