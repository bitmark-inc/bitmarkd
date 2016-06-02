// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package block

import (
	//"encoding/binary"
	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	//"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	//"time"
	"fmt"
)

func (pack Packed) Unpack() (*Block, error) {

	// see if able to contain a header
	if len(pack) < blockrecord.TotalBlockSize {
		return nil, fault.ErrInvalidBlock
	}

	blk := &Block{
		Header: blockrecord.New(),
	}

	// extract the header
	packedHeader := blockrecord.PackedHeader(pack[:blockrecord.TotalBlockSize])

	err := packedHeader.Unpack(blk.Header)
	if nil != err {
		return nil, err
	}

	// set timestamp
	// blk.Timestamp = time.Unix(blk.Header.Timestamp, 0).UTC()

	transactionCount := uint64(blk.Header.TransactionCount) // tx count is smaller that uint64
	// check if too big
	if transactionCount > MaximumTransactions {
		return nil, fault.ErrTooManyTransactionsInBlock
	}

	// unpack transactions
	packedTransactions := transactionrecord.Packed(pack[blockrecord.TotalBlockSize:])

	blk.TxIds = make([]merkle.Digest, transactionCount)
	blk.Transactions = make([]transactionrecord.Transaction, transactionCount)

	for i := uint64(0); i < transactionCount; i += 1 {
		var length int
		blk.Transactions[i], length, err = packedTransactions.Unpack()
		if nil != err {
			return nil, err
		}
		blk.TxIds[i] = merkle.NewDigest(packedTransactions[:length])
		packedTransactions = packedTransactions[length:]
	}

	// check that first transaction is a base record
	record, flag := transactionrecord.RecordName(blk.Transactions[0])
	if !flag || "BaseData" != record {
		fmt.Printf("1st: %q\n", record)
		fmt.Printf("1st: %#v\n", blk.Transactions)
		return nil, fault.ErrFirstTransactionIsNotBase
	}

	// merkle tree
	tree := merkle.FullMerkleTree(blk.TxIds)
	// treeBuffer := new(bytes.Buffer)
	// err = binary.Write(treeBuffer, binary.LittleEndian, tree)
	// fault.PanicIfError("block.Check - writing merkle", err)

	// if !bytes.Equal(treeBuffer.Bytes(), blk.Header.MerkleRoot) {
	// 	return nil, fault.ErrInvalidBlock
	// }

	// header checks
	blk.Digest = blockdigest.NewDigest(pack[:blockrecord.TotalBlockSize])

	if blk.Header.MerkleRoot != tree[len(tree)-1] {
		return nil, fault.ErrInvalidBlock // ***** FIX THIS: maybe different error
	}

	if blk.Digest.Cmp(blk.Header.Difficulty.BigInt()) > 0 {
		return nil, fault.ErrBlockHashDoesNotMeetDifficulty
	}

	return blk, nil
}
