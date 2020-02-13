// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package observer_test

import (
	"context"
	"crypto/rand"
	"testing"

	"github.com/bitmark-inc/bitmarkd/announce/mocks"
	"github.com/bitmark-inc/bitmarkd/announce/observer"
	"github.com/bitmark-inc/bitmarkd/announce/receptor"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/logger"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	ma "github.com/multiformats/go-multiaddr"
)

func TestSelfUpdate(t *testing.T) {
	ctl := gomock.NewController(t)
	m := mocks.NewMockReceptor(ctl)
	defer ctl.Finish()
	priv, _, _ := crypto.GenerateKeyPairWithReader(crypto.Ed25519, 0, rand.Reader)
	h, _ := libp2p.New(context.Background(), libp2p.Identity(priv))
	pID := h.ID()
	bID, _ := pID.Marshal()
	addr, _ := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/1234")
	bAddr, _ := proto.Marshal(&receptor.Addrs{Address: util.GetBytesFromMultiaddr([]ma.Multiaddr{addr})})

	m.EXPECT().SetSelf(pID, []ma.Multiaddr{addr}).Return(nil).Times(1)

	r := observer.NewSelf(m, logger.New(category))
	r.Update("self", [][]byte{bID, bAddr})
}

func TestSelfUpdateWhenEventNotMatch(t *testing.T) {
	ctl := gomock.NewController(t)
	m := mocks.NewMockReceptor(ctl)
	defer ctl.Finish()

	m.EXPECT().SetSelf(gomock.Any(), gomock.Any()).Return(nil).Times(0)

	r := observer.NewSelf(m, logger.New(category))
	r.Update("not_self", [][]byte{})
}
