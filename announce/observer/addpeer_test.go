// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package observer_test

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"testing"
	"time"

	p2pPeer "github.com/libp2p/go-libp2p-core/peer"

	"github.com/libp2p/go-libp2p"

	"github.com/bitmark-inc/bitmarkd/announce/receptor"
	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/crypto"

	"github.com/bitmark-inc/logger"

	"github.com/bitmark-inc/bitmarkd/util"

	ma "github.com/multiformats/go-multiaddr"

	"github.com/bitmark-inc/bitmarkd/announce/mocks"
	"github.com/bitmark-inc/bitmarkd/announce/observer"
	"github.com/golang/mock/gomock"
)

func TestAddpeerUpdate(t *testing.T) {
	ctl := gomock.NewController(t)
	m := mocks.NewMockReceptor(ctl)
	defer ctl.Finish()

	priv, _, _ := crypto.GenerateKeyPairWithReader(crypto.Ed25519, 0, rand.Reader)
	h, _ := libp2p.New(context.Background(), libp2p.Identity(priv))
	pID := h.ID()
	bID, _ := pID.Marshal()
	addr, _ := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/1234")
	bAddr, _ := proto.Marshal(&receptor.Addrs{Address: util.GetBytesFromMultiaddr([]ma.Multiaddr{addr})})
	now := uint64(time.Now().Unix())
	ts := make([]byte, 8)
	binary.BigEndian.PutUint64(ts, now)

	m.EXPECT().Add(pID, []ma.Multiaddr{addr}, now).Return(true).Times(1)

	r := observer.NewAddpeer(m, logger.New(category))
	r.Update("addpeer", [][]byte{bID, bAddr, ts})
}

func TestAddpeerUpdateWhenEventNotMatch(t *testing.T) {
	ctl := gomock.NewController(t)
	m := mocks.NewMockReceptor(ctl)
	defer ctl.Finish()

	m.EXPECT().Add(gomock.Any(), gomock.Any(), gomock.Any()).Return(true).Times(0)

	r := observer.NewAddpeer(m, logger.New(category))
	r.Update("not_addpeer", [][]byte{})
}

func TestAddpeerUpdateWhenIDError(t *testing.T) {
	ctl := gomock.NewController(t)
	m := mocks.NewMockReceptor(ctl)
	defer ctl.Finish()

	pID := p2pPeer.ID("test")
	bID, _ := pID.Marshal()

	r := observer.NewAddpeer(m, logger.New(category))
	r.Update("addpeer", [][]byte{bID})
}

func TestAddpeerUpdateWhenAddrsError(t *testing.T) {
	ctl := gomock.NewController(t)
	m := mocks.NewMockReceptor(ctl)
	defer ctl.Finish()

	priv, _, _ := crypto.GenerateKeyPairWithReader(crypto.Ed25519, 0, rand.Reader)
	h, _ := libp2p.New(context.Background(), libp2p.Identity(priv))
	pID := h.ID()
	bID, _ := pID.Marshal()
	now := uint64(time.Now().Unix())
	ts := make([]byte, 8)
	binary.BigEndian.PutUint64(ts, now)

	r := observer.NewAddpeer(m, logger.New(category))
	r.Update("addpeer", [][]byte{bID, []byte{1, 2, 3, 4}})
}

func TestAddpeerUpdateWhenAddrsZeroLength(t *testing.T) {
	ctl := gomock.NewController(t)
	m := mocks.NewMockReceptor(ctl)
	defer ctl.Finish()

	priv, _, _ := crypto.GenerateKeyPairWithReader(crypto.Ed25519, 0, rand.Reader)
	h, _ := libp2p.New(context.Background(), libp2p.Identity(priv))
	pID := h.ID()
	bID, _ := pID.Marshal()
	bAddr, _ := proto.Marshal(&receptor.Addrs{Address: [][]byte{}})
	now := uint64(time.Now().Unix())
	ts := make([]byte, 8)
	binary.BigEndian.PutUint64(ts, now)

	r := observer.NewAddpeer(m, logger.New(category))
	r.Update("addpeer", [][]byte{bID, bAddr})
}

func TestAddpeerUpdateWhenTimestampError(t *testing.T) {
	ctl := gomock.NewController(t)
	m := mocks.NewMockReceptor(ctl)
	defer ctl.Finish()

	priv, _, _ := crypto.GenerateKeyPairWithReader(crypto.Ed25519, 0, rand.Reader)
	h, _ := libp2p.New(context.Background(), libp2p.Identity(priv))
	pID := h.ID()
	bID, _ := pID.Marshal()
	addr, _ := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/1234")
	bAddr, _ := proto.Marshal(&receptor.Addrs{Address: util.GetBytesFromMultiaddr([]ma.Multiaddr{addr})})
	ts := make([]byte, 6)

	r := observer.NewAddpeer(m, logger.New(category))
	r.Update("addpeer", [][]byte{bID, bAddr, ts})
}
