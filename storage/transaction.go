// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package storage

import (
	"fmt"
	"sync"
)

// Transaction - concept from RDBMS
type Transaction interface {
	Abort()
	Begin() error
	Commit() error
	Delete(Handle, []byte)
	Get(Handle, []byte) []byte
	GetN(Handle, []byte) (uint64, bool)
	GetNB(Handle, []byte) (uint64, []byte)
	Has(Handle, []byte) bool
	InUse() bool
	Put(Handle, []byte, []byte, []byte)
	PutN(Handle, []byte, uint64)
}

// this interface contains too many behavior it doesn't need
// could it use "delegate pattern" for better abstraction

type TransactionData struct {
	sync.Mutex
	access []Access
}

func newTransaction(access []Access) Transaction {
	return &TransactionData{
		access: access,
	}
}

func (t *TransactionData) InUse() bool {
	for _, da := range t.access {
		if da.InUse() {
			return true
		}
	}
	return false
}

func (t *TransactionData) Begin() error {
	if t.InUse() {
		return fmt.Errorf("transaction already in use")
	}

	for _, access := range t.access {
		access.Begin()
	}

	return nil
}

func (t *TransactionData) Put(
	h Handle,
	key []byte,
	value []byte,
	additional []byte,
) {
	h.Put(key, value, additional)
}

func (t *TransactionData) PutN(h Handle, key []byte, value uint64) {
	h.PutN(key, value)
}

func (t *TransactionData) Delete(h Handle, key []byte) {
	h.Remove(key)
}

func (t *TransactionData) Commit() error {
	for _, access := range t.access {
		err := access.Commit()
		if nil != err {
			return err
		}
	}
	t.Abort()
	return nil
}

func (t *TransactionData) Get(h Handle, key []byte) []byte {
	return h.Get(key)
}

func (t *TransactionData) GetN(h Handle, key []byte) (uint64, bool) {
	num, found := h.GetN(key)
	return num, found
}

func (t *TransactionData) GetNB(h Handle, key []byte) (uint64, []byte) {
	num, buffer := h.GetNB(key)
	return num, buffer
}

func (t *TransactionData) Abort() {
	for _, da := range t.access {
		da.Abort()
	}
}

func (t *TransactionData) Has(h Handle, key []byte) bool {
	return h.Has(key)
}
