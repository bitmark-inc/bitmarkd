// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package observer_test

import (
	"testing"

	"github.com/bitmark-inc/logger"

	p2pPeer "github.com/libp2p/go-libp2p-core/peer"

	"github.com/bitmark-inc/bitmarkd/announce/mocks"
	"github.com/bitmark-inc/bitmarkd/announce/observer"
	"github.com/golang/mock/gomock"
)

func TestUpdatetimeUpdate(t *testing.T) {
	ctl := gomock.NewController(t)
	m := mocks.NewMockReceptor(ctl)
	defer ctl.Finish()
	myID := []byte("test")
	bID, _ := p2pPeer.IDFromBytes(myID)

	m.EXPECT().UpdateTime(bID, gomock.Any()).Return().Times(1)

	r := observer.NewUpdatetime(m, logger.New(category))
	r.Update("updatetime", [][]byte{myID})
}

func TestUpdatetimeUpdateWhenEventNotMatch(t *testing.T) {
	ctl := gomock.NewController(t)
	m := mocks.NewMockReceptor(ctl)
	defer ctl.Finish()

	m.EXPECT().UpdateTime(gomock.Any(), gomock.Any()).Return().Times(0)

	r := observer.NewUpdatetime(m, logger.New(category))
	r.Update("not_updatetime", [][]byte{})
}
