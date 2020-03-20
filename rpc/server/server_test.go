// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package server_test

import (
	"fmt"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"testing"

	"github.com/bitmark-inc/bitmarkd/rpc/share"

	"github.com/bitmark-inc/bitmarkd/rpc/transaction"

	"github.com/bitmark-inc/bitmarkd/rpc/blockowner"

	"github.com/bitmark-inc/bitmarkd/rpc/node"

	"github.com/bitmark-inc/bitmarkd/rpc/owner"

	"github.com/bitmark-inc/bitmarkd/pay"

	"github.com/bitmark-inc/bitmarkd/rpc/bitmarks"

	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/rpc/bitmark"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"

	"github.com/bitmark-inc/bitmarkd/fault"

	"github.com/bitmark-inc/bitmarkd/rpc/assets"

	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/bitmarkd/counter"

	"github.com/bitmark-inc/bitmarkd/rpc/server"
	"github.com/bitmark-inc/logger"

	"github.com/bitmark-inc/bitmarkd/rpc/fixtures"
)

var port string

func TestMain(m *testing.M) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	port = fmt.Sprintf(":%d", rand.Intn(30000)+30000) // 30,000 - 60,000
	c := counter.Counter(0)
	r := server.Create(logger.New(fixtures.LogCategory), "1.0", &c)
	l, _ := net.Listen("tcp", port)

	go r.Accept(l)
	r.HandleHTTP("/", "/debug")

	rc := m.Run()

	os.Exit(rc)
}

// following tests make sure proper methods are registered to server
// every test case error comes from specific method, this makes sures proper
// method is registered, but it also creates dependencies to specific function

func TestAssetsGet(t *testing.T) {
	conn, _ := net.Dial("tcp", port)
	defer conn.Close()

	client := rpc.NewClient(conn)
	defer client.Close()

	arg := assets.GetArguments{
		Fingerprints: []string{"test"},
	}
	var reply assets.GetReply
	err := client.Call("Assets.Get", &arg, &reply)
	assert.NotNil(t, err, "wrong Assets.Get")
	assert.Equal(t, fault.NotAvailableDuringSynchronise.Error(), err.Error(), "wrong reply")
}

func TestBitmarkTransfer(t *testing.T) {
	conn, _ := net.Dial("tcp", port)
	defer conn.Close()

	client := rpc.NewClient(conn)
	defer client.Close()

	arg := transactionrecord.BitmarkTransferCountersigned{
		Link:             merkle.Digest{},
		Escrow:           nil,
		Owner:            nil,
		Signature:        nil,
		Countersignature: nil,
	}
	var reply bitmark.TransferReply
	err := client.Call("Bitmark.Transfer", &arg, &reply)
	assert.NotNil(t, err, "wrong Bitmark.Transfer")
	assert.Equal(t, fault.InvalidItem.Error(), err.Error(), "wrong reply")
}

func TestBitmarkProvenance(t *testing.T) {
	conn, _ := net.Dial("tcp", port)
	defer conn.Close()

	client := rpc.NewClient(conn)
	defer client.Close()

	arg := bitmark.ProvenanceArguments{
		TxId:  merkle.Digest{},
		Count: 0,
	}
	var reply bitmark.ProvenanceReply
	err := client.Call("Bitmark.Provenance", &arg, &reply)
	assert.NotNil(t, err, "wrong Bitmark.Provenance")
	assert.Equal(t, fault.InvalidCount.Error(), err.Error(), "wrong reply")
}

func TestBitmarksCreate(t *testing.T) {
	conn, _ := net.Dial("tcp", port)
	defer conn.Close()

	client := rpc.NewClient(conn)
	defer client.Close()

	arg := bitmarks.CreateArguments{
		Assets: nil,
		Issues: nil,
	}
	var reply bitmarks.CreateReply
	err := client.Call("Bitmarks.Create", &arg, &reply)
	assert.NotNil(t, err, "wrong Bitmarks.Create")
	assert.Equal(t, fault.MissingParameters.Error(), err.Error(), "wrong reply")
}

func TestBitmarksProof(t *testing.T) {
	conn, _ := net.Dial("tcp", port)
	defer conn.Close()

	client := rpc.NewClient(conn)
	defer client.Close()

	arg := bitmarks.ProofArguments{
		PayId: pay.PayId{},
		Nonce: "",
	}
	var reply bitmarks.ProofReply
	err := client.Call("Bitmarks.Proof", &arg, &reply)
	assert.NotNil(t, err, "wrong Bitmarks.Proof")
	assert.Equal(t, fault.NotAvailableDuringSynchronise.Error(), err.Error(), "wrong reply")
}

func TestOwnerBitmarks(t *testing.T) {
	conn, _ := net.Dial("tcp", port)
	defer conn.Close()

	client := rpc.NewClient(conn)
	defer client.Close()

	arg := owner.BitmarksArguments{
		Owner: nil,
		Start: 0,
		Count: 0,
	}
	var reply owner.BitmarksReply
	err := client.Call("Owner.Bitmarks", &arg, &reply)
	assert.NotNil(t, err, "wrong Owner.Bitmarks")
	assert.Equal(t, fault.InvalidCount.Error(), err.Error(), "wrong reply")
}

func TestNodeList(t *testing.T) {
	conn, _ := net.Dial("tcp", port)
	defer conn.Close()

	client := rpc.NewClient(conn)
	defer client.Close()

	arg := node.Arguments{
		Start: 0,
		Count: 0,
	}
	var reply node.Reply
	err := client.Call("Node.List", &arg, &reply)
	assert.NotNil(t, err, "Node.List")
	assert.Equal(t, fault.InvalidCount.Error(), err.Error(), "wrong reply")
}

func TestNodeInfo(t *testing.T) {
	conn, _ := net.Dial("tcp", port)
	defer conn.Close()

	client := rpc.NewClient(conn)
	defer client.Close()

	arg := node.InfoArguments{}
	var reply node.InfoReply
	err := client.Call("Node.Info", &arg, &reply)
	assert.NotNil(t, err, "wrong Node.Info")
	assert.Equal(t, fault.DatabaseIsNotSet.Error(), err.Error(), "wrong node info")
}

func TestTransactionStatus(t *testing.T) {
	conn, _ := net.Dial("tcp", port)
	defer conn.Close()

	client := rpc.NewClient(conn)
	defer client.Close()

	arg := transaction.Arguments{
		TxId: merkle.Digest{},
	}

	var reply transaction.StatusReply
	err := client.Call("Transaction.Status", &arg, &reply)
	assert.NotNil(t, err, "Transaction.Status")
	assert.Equal(t, fault.MissingReservoir.Error(), err.Error(), "wrong reply")
}

func TestBlockOwnerTxIdForBlock(t *testing.T) {
	conn, _ := net.Dial("tcp", port)
	defer conn.Close()

	client := rpc.NewClient(conn)
	defer client.Close()

	arg := blockowner.TxIDForBlockArguments{
		BlockNumber: 0,
	}
	var reply blockowner.TxIDForBlockReply
	err := client.Call("BlockOwner.TxIDForBlock", &arg, &reply)
	assert.NotNil(t, err, "wrong BlockOwner.TxIDForBlock")
	assert.Equal(t, fault.DatabaseIsNotSet.Error(), err.Error(), "wrong reply")
}

func TestBlockOwnerTransfer(t *testing.T) {
	conn, _ := net.Dial("tcp", port)
	defer conn.Close()

	client := rpc.NewClient(conn)
	defer client.Close()

	arg := transactionrecord.BlockOwnerTransfer{
		Link:             merkle.Digest{},
		Escrow:           nil,
		Version:          0,
		Payments:         nil,
		Owner:            nil,
		Signature:        nil,
		Countersignature: nil,
	}
	var reply blockowner.TransferReply
	err := client.Call("BlockOwner.Transfer", &arg, &reply)
	assert.NotNil(t, err, "wrong BlockOwner.Transfer")
	assert.Equal(t, fault.NotAvailableDuringSynchronise.Error(), err.Error(), "wrong reply")
}

func TestShareCreate(t *testing.T) {
	conn, _ := net.Dial("tcp", port)
	defer conn.Close()

	client := rpc.NewClient(conn)
	defer client.Close()

	arg := transactionrecord.BitmarkShare{
		Link:      merkle.Digest{},
		Quantity:  0,
		Signature: nil,
	}
	var reply share.CreateReply
	err := client.Call("Share.Create", &arg, &reply)
	assert.NotNil(t, err, "wrong Share.Create")
	assert.Equal(t, fault.NotAvailableDuringSynchronise.Error(), err.Error(), "wrong reply")
}

func TestShareBalance(t *testing.T) {
	conn, _ := net.Dial("tcp", port)
	defer conn.Close()

	client := rpc.NewClient(conn)
	defer client.Close()

	arg := share.BalanceArguments{
		Owner:   nil,
		ShareId: merkle.Digest{},
		Count:   0,
	}
	var reply share.BalanceReply
	err := client.Call("Share.Balance", &arg, &reply)
	assert.NotNil(t, err, "Share.Balance")
	assert.Equal(t, fault.InvalidItem.Error(), err.Error(), "wrong reply")
}

func TestShareGrant(t *testing.T) {
	conn, _ := net.Dial("tcp", port)
	defer conn.Close()

	client := rpc.NewClient(conn)
	defer client.Close()

	arg := transactionrecord.ShareGrant{
		ShareId:          merkle.Digest{},
		Quantity:         0,
		Owner:            nil,
		Recipient:        nil,
		BeforeBlock:      0,
		Signature:        nil,
		Countersignature: nil,
	}
	var reply share.GrantReply
	err := client.Call("Share.Grant", &arg, &reply)
	assert.NotNil(t, err, "Share.Grant")
	assert.Equal(t, fault.InvalidItem.Error(), err.Error(), "wrong reply")
}
