package rpc_test

import (
	"crypto/ed25519"
	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/chain"
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/difficulty"
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

func TestBitmarksCreate(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()

	mode.Initialise(chain.Testing)
	defer mode.Finalise()

	bus := messagebus.Bus.P2P.Chan()
	defer messagebus.Bus.P2P.Release()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	poolA := mocks.NewMockHandle(ctl)
	poolB := mocks.NewMockHandle(ctl)
	r := mocks.NewMockReservoir(ctl)

	b := rpc.Bitmarks{
		Log:                   logger.New(logCategory),
		Limiter:               rate.NewLimiter(100, 100),
		IsNormalMode:          func(_ mode.Mode) bool { return true },
		Rsvr:                  r,
		PoolAssets:            poolA,
		PoolBlockOwnerPayment: poolB,
	}

	acc := account.Account{
		AccountInterface: &account.ED25519Account{
			Test:      true,
			PublicKey: issuerPublicKey,
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
	ad.Signature = ed25519.Sign(issuerPrivateKey, packed)
	packed, _ = ad.Pack(&acc)
	aid := ad.AssetId()

	is := transactionrecord.BitmarkIssue{
		AssetId:   aid,
		Owner:     &acc,
		Nonce:     1,
		Signature: nil,
	}
	packed2, _ := is.Pack(&acc)
	is.Signature = ed25519.Sign(issuerPrivateKey, packed2)
	packed2, _ = is.Pack(&acc)

	arg := rpc.CreateArguments{
		Assets: []*transactionrecord.AssetData{
			&ad,
		},
		Issues: []*transactionrecord.BitmarkIssue{
			&is,
		},
	}

	info := reservoir.IssueInfo{
		TxIds:      []merkle.Digest{merkle.Digest{1, 2, 3, 4}},
		Packed:     []byte{1, 2, 3, 4},
		Id:         pay.PayId{1, 2, 3, 4},
		Nonce:      reservoir.PayNonce{4, 3, 2, 1},
		Difficulty: difficulty.New(),
		Payments: []transactionrecord.PaymentAlternative{
			[]*transactionrecord.Payment{
				&transactionrecord.Payment{
					Currency: currency.Litecoin,
					Address:  litecoinAddress,
					Amount:   100,
				},
			},
		},
	}

	poolA.EXPECT().Has(aid[:]).Return(true).Times(1)
	r.EXPECT().StoreIssues(gomock.Any()).Return(&info, false, nil).Times(1)

	var reply rpc.CreateReply
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
