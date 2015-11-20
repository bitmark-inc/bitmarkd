// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package peer

import (
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/transaction"
	"github.com/bitmark-inc/logger"
)

type Transaction struct {
	log *logger.L
}

// ------------------------------------------------------------

type TransactionPutArguments struct {
	Bilateral_SENDER string // magick field
	Tx               transaction.Packed
}

type TransactionPutReply struct {
	Duplicate bool
}

// new incoming transaction
func (t *Transaction) Put(arguments *TransactionPutArguments, reply *TransactionPutReply) error {

	packedTx := arguments.Tx
	from := arguments.Bilateral_SENDER

	t.log.Debugf("received from: %q tx: %x", from, packedTx)

	_, reply.Duplicate = packedTx.Exists()

	// propagate
	if !reply.Duplicate {
		t.log.Tracef("propagate: Tx = %x", packedTx)
		messagebus.Send(from, packedTx)
	}

	return nil
}

// ------------------------------------------------------------

type TransactionGetArguments struct {
	TxId transaction.Link
}

type TransactionGetReply struct {
	State transaction.State
	Data  []byte
}

// read a specific transaction
func (t *Transaction) Get(arguments *TransactionGetArguments, reply *TransactionGetReply) error {
	state, data, found := arguments.TxId.Read()
	if !found {
		t.log.Warnf("Get: does not exist: txId: %#v", arguments.TxId)
		return fault.ErrLinkNotFound
	}
	t.log.Infof("Get: state: %v  data: %x  txId: %#v", state, data[:8], arguments.TxId)
	reply.State = state
	reply.Data = data
	return nil
}
