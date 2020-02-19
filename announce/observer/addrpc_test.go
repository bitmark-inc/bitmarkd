// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package observer_test

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/bitmark-inc/logger"

	"github.com/bitmark-inc/bitmarkd/announce/mocks"
	"github.com/bitmark-inc/bitmarkd/announce/observer"
	"github.com/golang/mock/gomock"
)

func TestAddrpcUpdate(t *testing.T) {
	ctl := gomock.NewController(t)
	m := mocks.NewMockRPC(ctl)
	defer ctl.Finish()
	bfp := make([]byte, 32)
	bfp[0] = 1
	bfp[1] = 2
	b := []byte{5, 6, 7, 8}
	now := uint64(time.Now().Unix())
	ts := make([]byte, 8)
	binary.BigEndian.PutUint64(ts, now)

	m.EXPECT().Add(bfp, b, now).Return(true).Times(1)

	r := observer.NewAddrpc(m, logger.New(category))
	r.Update("addrpc", [][]byte{bfp, b, ts})
}
