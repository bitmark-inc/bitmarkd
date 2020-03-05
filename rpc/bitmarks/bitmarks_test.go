package bitmarks_test

import (
	"crypto/ed25519"
	"testing"

	"github.com/bitmark-inc/bitmarkd/rpc/bitmarks"

	"github.com/bitmark-inc/bitmarkd/rpc/fixtures"

	"github.com/bitmark-inc/bitmarkd/blockrecord"

	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/chain"
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/bitmarkd/rpc/mocks"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestBitmarksCreate(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	mode.Initialise(chain.Testing)
	defer mode.Finalise()

	bus := messagebus.Bus.P2P.Chan()
	defer messagebus.Bus.P2P.Release()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	poolA := mocks.NewMockHandle(ctl)
	poolB := mocks.NewMockHandle(ctl)
	r := mocks.NewMockReservoir(ctl)

	b := bitmarks.New(
		logger.New(fixtures.LogCategory),
		reservoir.Handles{
			Assets:            poolA,
			BlockOwnerPayment: poolB,
		},
		func(_ mode.Mode) bool { return true },
		r,
	)

	acc := account.Account{
		AccountInterface: &account.ED25519Account{
			Test:      true,
			PublicKey: fixtures.IssuerPublicKey,
		},
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
	aid := ad.AssetId()

	is := transactionrecord.BitmarkIssue{
		AssetId:   aid,
		Owner:     &acc,
		Nonce:     1,
		Signature: nil,
	}

	arg := bitmarks.CreateArguments{
		Assets: []*transactionrecord.AssetData{
			&ad,
		},
		Issues: []*transactionrecord.BitmarkIssue{
			&is,
		},
	}

	info := reservoir.IssueInfo{
		TxIds:      []merkle.Digest{{1, 2, 3, 4}},
		Packed:     []byte{1, 2, 3, 4},
		Id:         pay.PayId{1, 2, 3, 4},
		Nonce:      reservoir.PayNonce{4, 3, 2, 1},
		Difficulty: difficulty.New(),
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

	poolA.EXPECT().Has(aid[:]).Return(true).Times(1)
	r.EXPECT().StoreIssues(gomock.Any()).Return(&info, false, nil).Times(1)

	var reply bitmarks.CreateReply
	err := b.Create(&arg, &reply)
	assert.Nil(t, err, "wrong Create")

	assert.Equal(t, 1, len(reply.Assets), "wrong asset count")
	assert.Equal(t, aid.String(), reply.Assets[0].AssetId.String(), "wrong asset id")

	assert.Equal(t, 1, len(reply.Issues), "wrong issue count")
	assert.Equal(t, info.TxIds[0], reply.Issues[0].TxId, "wrong tx id")
	assert.Equal(t, info.Id, reply.PayId, "wrong pay id")
	assert.Equal(t, info.Nonce, reply.PayNonce, "wrong pay nonce")
	assert.Equal(t, difficulty.New().GoString(), reply.Difficulty, "wrong difficulty")
	assert.Equal(t, info.Payments[0][0].Address, reply.Payments[currency.Litecoin.String()][0].Address, "wrong payments")

	msg := <-bus
	assert.Equal(t, "issues", msg.Command, "wrong command")
}

func TestBitmarksProof(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	mode.Initialise(chain.Testing)
	defer mode.Finalise()

	bus := messagebus.Bus.P2P.Chan()
	defer messagebus.Bus.P2P.Release()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	poolA := mocks.NewMockHandle(ctl)
	poolB := mocks.NewMockHandle(ctl)
	r := mocks.NewMockReservoir(ctl)

	b := bitmarks.New(
		logger.New(fixtures.LogCategory),
		reservoir.Handles{
			Assets:            poolA,
			BlockOwnerPayment: poolB,
		},
		func(_ mode.Mode) bool { return true },
		r,
	)

	nonce := blockrecord.NonceType(0x1234567890abcdef)
	nonceBytes, _ := nonce.MarshalText()

	arg := bitmarks.ProofArguments{
		PayId: pay.PayId{},
		Nonce: string(nonceBytes),
	}

	r.EXPECT().TryProof(gomock.Any(), gomock.Any()).Return(reservoir.TrackingAccepted).Times(1)

	var reply bitmarks.ProofReply
	err := b.Proof(&arg, &reply)
	assert.Nil(t, err, "wrong Create")

	msg := <-bus
	assert.Equal(t, "proof", msg.Command, "wrong command")
}
