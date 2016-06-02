// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package block

import (
	//"encoding/binary"
	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	//"time"
)

// limit the maximum transactions in a block
const (
	MaximumTransactions = 20000
)

// type to hold the unpacked block
type Block struct {
	Header *blockrecord.Header
	Digest blockdigest.Digest
	//Timestamp    time.Time
	TxIds        []merkle.Digest
	Transactions []transactionrecord.Transaction
}

// for packed block
// structure:  <packed-header>[<packed-txn>,<packed-txn,...]
type Packed []byte // for general distribution
