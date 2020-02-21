// Use of this source code is governed by an ISC
// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// license that can be found in the LICENSE file.

package rpc_test

import (
	"testing"
	"time"

	"github.com/bitmark-inc/bitmarkd/announce/fingerprint"
	announceRPC "github.com/bitmark-inc/bitmarkd/announce/rpc"
	"github.com/bitmark-inc/bitmarkd/util"

	"github.com/bitmark-inc/bitmarkd/rpc/mocks"

	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/logger"
	"golang.org/x/time/rate"

	"github.com/bitmark-inc/bitmarkd/rpc"
)

func TestNode_List(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	a := mocks.NewMockAnnounce(ctl)

	now := time.Now()
	n := rpc.Node{
		Log:      logger.New(logCategory),
		Limiter:  rate.NewLimiter(100, 100),
		Start:    now,
		Version:  "1",
		Announce: a,
	}

	arg := rpc.NodeArguments{
		Start: 100,
		Count: 5,
	}

	c1, _ := util.NewConnection("1.2.3.4:1234")

	entry := announceRPC.Entry{
		Fingerprint: fingerprint.Type{1, 2, 3, 4},
		Connections: []*util.Connection{c1},
	}

	a.EXPECT().Fetch(arg.Start, arg.Count).Return([]announceRPC.Entry{entry}, uint64(10), nil).Times(1)

	var reply rpc.NodeReply
	err := n.List(&arg, &reply)
	assert.Nil(t, err, "wrong List")
	assert.Equal(t, 1, len(reply.Nodes), "wrong node count")
	assert.Equal(t, entry, reply.Nodes[0], "wrong node info")
	assert.Equal(t, uint64(10), reply.NextStart, "wrong next start")
}
