// Use of this source code is governed by an ISC
// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// license that can be found in the LICENSE file.

package node_test

import (
	"testing"
	"time"

	"github.com/bitmark-inc/bitmarkd/reservoir"

	"github.com/bitmark-inc/bitmarkd/counter"

	"github.com/bitmark-inc/bitmarkd/rpc/fixtures"

	"github.com/bitmark-inc/bitmarkd/storage"

	"github.com/bitmark-inc/bitmarkd/chain"
	"github.com/bitmark-inc/bitmarkd/mode"

	"github.com/bitmark-inc/bitmarkd/announce/fingerprint"
	"github.com/bitmark-inc/bitmarkd/announce/rpc"
	"github.com/bitmark-inc/bitmarkd/util"

	"github.com/bitmark-inc/bitmarkd/rpc/mocks"

	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/bitmarkd/rpc/node"
	"github.com/bitmark-inc/logger"
)

func TestNode_List(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	a := mocks.NewMockAnnounce(ctl)

	now := time.Now()
	ctr := counter.Counter(3)
	n := node.New(
		logger.New(fixtures.LogCategory),
		reservoir.Handles{},
		now,
		"1",
		&ctr,
		a,
	)

	arg := node.NodeArguments{
		Start: 100,
		Count: 5,
	}

	c1, _ := util.NewConnection("1.2.3.4:1234")

	entry := rpc.Entry{
		Fingerprint: fingerprint.Type{1, 2, 3, 4},
		Connections: []*util.Connection{c1},
	}

	a.EXPECT().Fetch(arg.Start, arg.Count).Return([]rpc.Entry{entry}, uint64(10), nil).Times(1)

	var reply node.NodeReply
	err := n.List(&arg, &reply)
	assert.Nil(t, err, "wrong List")
	assert.Equal(t, 1, len(reply.Nodes), "wrong node count")
	assert.Equal(t, entry, reply.Nodes[0], "wrong node info")
	assert.Equal(t, uint64(10), reply.NextStart, "wrong next Start")
}

func TestNodeInfo(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	mode.Initialise(chain.Testing)
	defer mode.Finalise()

	a := mocks.NewMockAnnounce(ctl)
	b := mocks.NewMockHandle(ctl)

	now := time.Now()
	c := counter.Counter(5)

	n := node.New(
		logger.New(fixtures.LogCategory),
		reservoir.Handles{
			Blocks: b,
		},
		now,
		"100",
		&c,
		a,
	)

	b.EXPECT().LastElement().Return(storage.Element{}, false).Times(1)

	var reply node.InfoReply
	err := n.Info(&node.InfoArguments{}, &reply)
	assert.Nil(t, err, "wrong Info")
	assert.Equal(t, chain.Testing, reply.Chain, "wrong chain")
	assert.Equal(t, mode.Resynchronise.String(), reply.Mode, "wrong mode")
	assert.Equal(t, uint64(0), reply.Block.Height, "wrong block height")
	assert.Equal(t, "", reply.Block.Hash, "wrong block hash")
	assert.Equal(t, uint64(0), reply.Miner.Success, "wrong success mined")
	assert.Equal(t, uint64(0), reply.Miner.Failed, "wrong failed mined")
	assert.Equal(t, c.Uint64(), reply.RPCs, "wrong connection count")
	assert.Equal(t, n.Version, reply.Version, "wrong version")
	assert.Equal(t, "", reply.PublicKey, "wrong empty public key")
}
