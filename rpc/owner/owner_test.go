// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package owner_test

import (
	"crypto/ed25519"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/chain"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/ownership"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/bitmarkd/rpc/fixtures"
	"github.com/bitmark-inc/bitmarkd/rpc/mocks"
	"github.com/bitmark-inc/bitmarkd/rpc/owner"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
)

func TestOwnerBitmarks(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	mode.Initialise(chain.Testing)
	defer mode.Finalise()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	tr := mocks.NewMockHandle(ctl)
	a := mocks.NewMockHandle(ctl)
	os := mocks.NewMockOwnership(ctl)

	o := owner.New(
		logger.New(fixtures.LogCategory),
		reservoir.Handles{
			Assets:       a,
			Transactions: tr,
		},
		os,
	)

	acc := account.Account{
		AccountInterface: &account.ED25519Account{
			Test:      true,
			PublicKey: fixtures.IssuerPublicKey,
		},
	}

	arg := owner.BitmarksArguments{
		Owner: &acc,
		Start: 5,
		Count: 10,
	}

	n := uint64(3)
	ass := transactionrecord.NewAssetIdentifier([]byte{1, 2, 3, 4})

	r := ownership.Record{
		N:           1,
		TxId:        merkle.Digest{},
		IssueTxId:   merkle.Digest{},
		Item:        ownership.OwnedAsset,
		AssetId:     &ass,
		BlockNumber: &n,
	}

	ad := transactionrecord.AssetData{
		Name:        "test",
		Fingerprint: "fingerprint",
		Metadata:    "owner\x00me",
		Registrant:  &acc,
		Signature:   nil,
	}
	packed, _ := ad.Pack(&acc)
	ad.Signature = ed25519.Sign(fixtures.IssuerPrivateKey, packed)
	packed, _ = ad.Pack(&acc)

	os.EXPECT().ListBitmarksFor(arg.Owner, arg.Start, arg.Count).Return([]ownership.Record{r}, nil).Times(1)
	tr.EXPECT().GetNB(r.TxId[:]).Return(uint64(1), packed).Times(1)
	a.EXPECT().GetNB(r.AssetId[:]).Return(uint64(1), packed).Times(1)

	var reply owner.BitmarksReply
	err := o.Bitmarks(&arg, &reply)
	assert.Nil(t, err, "wrong Bitmarks")
	assert.Equal(t, r.N+1, reply.Next, "wrong next")
	assert.Equal(t, 1, len(reply.Data), "wrong record count")
	assert.Equal(t, r, reply.Data[0], "wrong asset")
	assert.Equal(t, 2, len(reply.Tx), "wrong tx count")
	assert.Equal(t, ad, *reply.Tx[r.TxId.String()].Data.(*transactionrecord.AssetData), "wrong first record")
	assert.Equal(t, ad, *reply.Tx[r.TxId.String()].Data.(*transactionrecord.AssetData))
}
