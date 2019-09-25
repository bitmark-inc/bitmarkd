// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package payment

import (
	"reflect"
	"testing"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/peer"
	"github.com/btcsuite/btcd/wire"
	"golang.org/x/sync/errgroup"

	"github.com/bitmark-inc/bitmarkd/currency"
)

func NewDummyMsgBlock(previousBlock *chainhash.Hash, timestamp *time.Time) *wire.MsgBlock {
	block := wire.NewMsgBlock(wire.NewBlockHeader(1, &chainhash.Hash{}, &chainhash.Hash{}, 1, 1))

	if previousBlock != nil {
		block.Header.PrevBlock = *previousBlock
	}

	if timestamp != nil {
		block.Header.Timestamp = *timestamp
	} else {
		block.Header.Timestamp = time.Unix(1317972665, 0)
	}

	return block
}

func TestOnPeerBlockEarlyBlocks(t *testing.T) {
	testCurrency := currency.Bitcoin

	w, err := newP2pWatcher(testCurrency)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	p, err := peer.NewOutboundPeer(w.peerConfig(), "127.0.0.1:12345")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	blockMsg := NewDummyMsgBlock(nil, nil)
	err = w.onPeerBlock(p, blockMsg, nil)

	if err != ErrBlockIsTooOld {
		t.Fatalf("error is not what we expected. expected: %s, actual: %s", ErrBlockIsTooOld, err)
	}
}

func TestOnPeerBlockHeaderNotFound(t *testing.T) {
	testCurrency := currency.Bitcoin

	w, err := newP2pWatcher(testCurrency)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	p, err := peer.NewOutboundPeer(w.peerConfig(), "127.0.0.1:12345")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	now := time.Now()
	blockMsg := NewDummyMsgBlock(nil, &now)

	err = w.onPeerBlock(p, blockMsg, nil)

	if err != ErrBlockHeaderNotFound {
		t.Fatalf("error is not what we expected. expected: %s, actual: %s", ErrBlockHeaderNotFound, err)
	}
}

func TestOnPeerBlockProcessed(t *testing.T) {
	testCurrency := currency.Bitcoin

	w, err := newP2pWatcher(testCurrency)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	p, err := peer.NewOutboundPeer(w.peerConfig(), "127.0.0.1:12345")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	checkpoint := testCurrency.ChainParam(true).Checkpoints[0]

	// prepare checkpoint header in db
	w.storage.StoreBlock(checkpoint.Height, checkpoint.Hash)

	// prepare this block header in db
	now := time.Now()
	blockMsg := NewDummyMsgBlock(checkpoint.Hash, &now)
	blockHash := blockMsg.BlockHash()
	w.storage.StoreBlock(checkpoint.Height+1, &blockHash)

	w.blockCache.Set(blockHash.String(), true, 0)

	err = w.onPeerBlock(p, blockMsg, nil)

	if err != ErrBlockAlreadyProcessed {
		t.Fatalf("error is not what we expected. expected: %s, actual: %s", ErrBlockAlreadyProcessed, err)
	}
}

func TestExamineTx(t *testing.T) {
	payIdByte := []byte{0x37, 0xa3, 0x80, 0x0e, 0x22, 0x2f, 0x1f, 0xa1,
		0x1c, 0x31, 0x34, 0xab, 0xfd, 0x6c, 0xcf, 0x9c, 0xc9, 0xe7,
		0x61, 0x78, 0x35, 0x1d, 0xb2, 0xa2, 0x76, 0x5d, 0xbb, 0x60,
		0xe4, 0x65, 0x9d, 0x35, 0x2c, 0x5d, 0x91, 0x11, 0xc3, 0x29,
		0x38, 0x04, 0x1f, 0x7b, 0x98, 0x67, 0xe1, 0xaf, 0x91, 0x1f,
	}
	paidAddress := "mzkCaHJmu1gdnsL9jxW2bwqtw2MCCy66Ds"
	var paidAmount uint64 = 10000

	tx := &wire.MsgTx{
		Version: 1,
		TxIn:    nil,
		TxOut: []*wire.TxOut{
			{
				Value: int64(paidAmount),
				PkScript: []byte{
					0x76, 0xa9, 0x14, 0xd2, 0xeb, 0xb7, 0xb2, 0x59, 0xfb, 0x74,
					0x10, 0xdc, 0xa1, 0x9b, 0x70, 0x7c, 0x40, 0x91, 0x19, 0x5d,
					0x81, 0x8a, 0xc4, 0x88, 0xac,
				},
			},
			{
				Value:    0,
				PkScript: append([]byte{0x6a, 0x30}, payIdByte...),
			},
		},
		LockTime: 0,
	}

	testCurrency := currency.Litecoin

	w, err := newP2pWatcher(testCurrency)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	b, amounts := w.examineTransaction(tx)

	if !reflect.DeepEqual(payIdByte, b) {
		t.Fatalf("unexpected payment id. expect: %v, actual: %v", payIdByte, b)

	}

	v, ok := amounts[paidAddress]
	if !ok {
		t.Fatalf("the paid address: %s is not appeared", paidAddress)
	}

	if v != paidAmount {
		t.Fatalf("unexpected amount for the payment address. expected: %d, actual: %d", paidAmount, v)
	}
}

func TestExamineTxWithoutPayment(t *testing.T) {
	var paidAmount uint64 = 10000

	tx := &wire.MsgTx{
		Version: 1,
		TxIn:    nil,
		TxOut: []*wire.TxOut{
			{
				Value: int64(paidAmount),
				PkScript: []byte{
					0x76, 0xa9, 0x14, 0xd2, 0xeb, 0xb7, 0xb2, 0x59, 0xfb, 0x74,
					0x10, 0xdc, 0xa1, 0x9b, 0x70, 0x7c, 0x40, 0x91, 0x19, 0x5d,
					0x81, 0x8a, 0xc4, 0x88, 0xac,
				},
			},
		},
		LockTime: 0,
	}

	testCurrency := currency.Litecoin

	w, err := newP2pWatcher(testCurrency)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	b, _ := w.examineTransaction(tx)

	if b != nil {
		t.Fatalf("expect the pay id be nil. actual: %v", b)
	}
}

func TestOnPeerNoHeaders(t *testing.T) {
	testCurrency := currency.Litecoin

	w, err := newP2pWatcher(testCurrency)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	p, err := peer.NewOutboundPeer(w.peerConfig(), "127.0.0.1:12345")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	headers := wire.NewMsgHeaders()

	var g errgroup.Group

	g.Go(func() error { return <-w.onHeadersErr })
	w.onPeerHeaders(p, headers)

	if err := g.Wait(); err != ErrNoNewBlockHeadersFromPeer {
		t.Fatalf("unexpected error. expect: %s, actual: %s", ErrNoNewBlockHeadersFromPeer, err)
	}
}

func TestOnPeerAllOldHeaders(t *testing.T) {
	testCurrency := currency.Litecoin

	w, err := newP2pWatcher(testCurrency)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	fakeHeader1 := wire.NewBlockHeader(1, &chainhash.Hash{}, &chainhash.Hash{}, 1, 1)
	fakeHash1 := fakeHeader1.BlockHash()
	fakeHeader2 := wire.NewBlockHeader(1, &fakeHash1, &chainhash.Hash{}, 1, 1)
	fakeHash2 := fakeHeader2.BlockHash()

	if err := w.storage.StoreBlock(1, &fakeHash1); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if err := w.storage.StoreBlock(2, &fakeHash2); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	w.lastHash = &fakeHash2
	w.lastHeight = 2

	p, err := peer.NewOutboundPeer(w.peerConfig(), "127.0.0.1:12345")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	headers := wire.NewMsgHeaders()
	headers.Headers = append(headers.Headers, fakeHeader1, fakeHeader2)

	var g errgroup.Group

	g.Go(func() error { return <-w.onHeadersErr })
	w.onPeerHeaders(p, headers)

	if err := g.Wait(); err != ErrNoNewBlockHeadersFromPeer {
		t.Fatalf("unexpected error. expect: %s, actual: %s", ErrNoNewBlockHeadersFromPeer, err)
	}
}

func TestOnPeerInvalidPrevious(t *testing.T) {
	testCurrency := currency.Litecoin

	w, err := newP2pWatcher(testCurrency)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	fakeHeader1 := wire.NewBlockHeader(1, &chainhash.Hash{}, &chainhash.Hash{}, 1, 1)
	fakeHash1 := fakeHeader1.BlockHash()
	wrongPrevHash := &chainhash.Hash{}
	// make the hash different from the first one
	wrongPrevHash[0] = 1
	fakeHeader2 := wire.NewBlockHeader(1, wrongPrevHash, &chainhash.Hash{}, 1, 1)

	if err := w.storage.StoreBlock(1, &fakeHash1); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	w.lastHash = &fakeHash1
	w.lastHeight = 1

	p, err := peer.NewOutboundPeer(w.peerConfig(), "127.0.0.1:12345")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	headers := wire.NewMsgHeaders()
	headers.Headers = append(headers.Headers, fakeHeader1, fakeHeader2)

	var g errgroup.Group

	g.Go(func() error { return <-w.onHeadersErr })
	w.onPeerHeaders(p, headers)

	if err := g.Wait(); err != ErrMissingPreviousBlockHeader {
		t.Fatalf("unexpected error. expect: %s, actual: %s", ErrMissingPreviousBlockHeader, err)
	}
}

func TestRollbackToHeight(t *testing.T) {
	testCurrency := currency.Litecoin

	w, err := newP2pWatcher(testCurrency)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	fakeHeader1 := wire.NewBlockHeader(99997997, &chainhash.Hash{}, &chainhash.Hash{}, 1, 1)
	fakeHash1 := fakeHeader1.BlockHash()
	fakeHeader2 := wire.NewBlockHeader(99997998, &fakeHash1, &chainhash.Hash{}, 1, 1)
	fakeHash2 := fakeHeader2.BlockHash()
	fakeHeader3 := wire.NewBlockHeader(99997999, &fakeHash2, &chainhash.Hash{}, 1, 1)
	fakeHash3 := fakeHeader3.BlockHash()

	w.lastHeight = 99999997

	if err := w.storage.StoreBlock(99997997, &fakeHash1); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if err := w.storage.StoreBlock(99997998, &fakeHash2); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if err := w.storage.StoreBlock(99997999, &fakeHash3); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if err := w.rollbackBlock(); err != nil {
		t.Fatalf("unexpected error. expected nil, actual: %s", err)
	}

	if w.lastHeight != 99997997 {
		t.Fatalf("unexpected amount for the payment address. expected: %d, actual: %d", 9997997, w.lastHeight)
	}

	if !reflect.DeepEqual(w.lastHash, &fakeHash1) {
		t.Fatalf("unexpected last hash. expected: %d, actual: %d", &fakeHash1, w.lastHash)
	}
}
