// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc_test

import (
	"encoding/binary"
	"testing"

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
