// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package assets_test

import (
	"crypto/ed25519"
	"testing"

	"github.com/bitmark-inc/bitmarkd/chain"

	"github.com/bitmark-inc/bitmarkd/reservoir"

	"github.com/bitmark-inc/bitmarkd/rpc/assets"

	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/rpc/fixtures"
	"github.com/bitmark-inc/bitmarkd/rpc/mocks"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestAssetsGet(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	mode.Initialise(chain.Testing)
	defer mode.Finalise()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	p := mocks.NewMockHandle(ctl)

	a := assets.New(
		logger.New(fixtures.LogCategory),
		reservoir.Handles{
			Assets: p,
		},
		func(_ mode.Mode) bool { return true },
		mode.IsTesting,
	)

	arg := assets.GetArguments{Fingerprints: []string{"fin1", "fin2"}}
	var reply assets.GetReply
	bin1 := transactionrecord.NewAssetIdentifier([]byte("fin1"))
	bin2 := transactionrecord.NewAssetIdentifier([]byte("fin2"))
	acc := &account.Account{
		AccountInterface: &account.ED25519Account{
			Test:      true,
			PublicKey: fixtures.IssuerPublicKey,
		},
	}
	ad := transactionrecord.AssetData{
		Name:        "test",
		Fingerprint: "123456789",
		Metadata:    "owner\x00test",
		Registrant:  acc,
	}
	packed, _ := ad.Pack(acc)
	signature := ed25519.Sign(fixtures.IssuerPrivateKey, packed)
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
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	mode.Initialise(chain.Testing)
	defer mode.Finalise()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	p := mocks.NewMockHandle(ctl)

	a := assets.New(
		logger.New(fixtures.LogCategory),
		reservoir.Handles{
			Assets: p,
		},
		func(_ mode.Mode) bool { return false },
		mode.IsTesting,
	)

	var reply assets.GetReply
	arg := assets.GetArguments{Fingerprints: []string{"fin1", "fin2"}}

	err := a.Get(&arg, &reply)
	assert.Equal(t, fault.NotAvailableDuringSynchronise, err, "wrong error")
}

func TestAssetsGetWhenNilAsset(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	p := mocks.NewMockHandle(ctl)

	a := assets.New(
		logger.New(fixtures.LogCategory),
		reservoir.Handles{
			Assets: p,
		},
		func(_ mode.Mode) bool { return true },
		mode.IsTesting,
	)

	arg := assets.GetArguments{Fingerprints: []string{"fin1"}}
	var reply assets.GetReply
	bin1 := transactionrecord.NewAssetIdentifier([]byte("fin1"))
	acc := &account.Account{
		AccountInterface: &account.ED25519Account{
			Test:      true,
			PublicKey: fixtures.IssuerPublicKey,
		},
	}
	ad := transactionrecord.AssetData{
		Name:        "test",
		Fingerprint: "123456789",
		Metadata:    "owner\x00test",
		Registrant:  acc,
	}
	packed, _ := ad.Pack(acc)
	signature := ed25519.Sign(fixtures.IssuerPrivateKey, packed)
	ad.Signature = signature
	packed, _ = ad.Pack(acc)

	p.EXPECT().GetNB(bin1[:]).Return(uint64(1), nil).Times(1)

	err := a.Get(&arg, &reply)
	assert.Nil(t, err, "wrong get")
	assert.Equal(t, 1, len(reply.Assets), "wrong asset count")

	assert.False(t, reply.Assets[0].Confirmed, "wrong confirmed")
}

func TestAssetsGetWhenUnpackError(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	p := mocks.NewMockHandle(ctl)

	a := assets.New(
		logger.New(fixtures.LogCategory),
		reservoir.Handles{
			Assets: p,
		},
		func(_ mode.Mode) bool { return true },
		mode.IsTesting,
	)

	arg := assets.GetArguments{Fingerprints: []string{"fin1", "fin2"}}
	var reply assets.GetReply
	bin1 := transactionrecord.NewAssetIdentifier([]byte("fin1"))
	bin2 := transactionrecord.NewAssetIdentifier([]byte("fin2"))
	acc := &account.Account{
		AccountInterface: &account.ED25519Account{
			Test:      true,
			PublicKey: fixtures.IssuerPublicKey,
		},
	}
	ad := transactionrecord.AssetData{
		Name:        "test",
		Fingerprint: "123456789",
		Metadata:    "owner\x00test",
		Registrant:  acc,
	}
	packed, _ := ad.Pack(acc)
	signature := ed25519.Sign(fixtures.IssuerPrivateKey, packed)
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

func TestRegister(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	mode.Initialise(chain.Testing)
	defer mode.Finalise()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	p := mocks.NewMockHandle(ctl)

	acc := &account.Account{
		AccountInterface: &account.ED25519Account{
			Test:      true,
			PublicKey: fixtures.IssuerPublicKey,
		},
	}
	ad := transactionrecord.AssetData{
		Name:        "test",
		Fingerprint: "123456789",
		Metadata:    "owner\x00test",
		Registrant:  acc,
	}
	packed, _ := ad.Pack(acc)
	signature := ed25519.Sign(fixtures.IssuerPrivateKey, packed)
	ad.Signature = signature
	packed, _ = ad.Pack(acc)

	p.EXPECT().Has(gomock.Any()).Return(true).Times(1)

	status, data, err := assets.Register([]*transactionrecord.AssetData{&ad}, p)
	assert.Nil(t, err, "wrong Register")
	assert.NotNil(t, data, "wrong data")
	assert.Equal(t, 1, len(status), "wrong status count")
	assert.Equal(t, true, status[0].Duplicate, "wrong duplicate status")
	assert.Equal(t, ad.AssetId(), *status[0].AssetId, "wrong asset ID")
}
