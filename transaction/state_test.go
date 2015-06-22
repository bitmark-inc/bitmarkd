// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package transaction_test

import (
	"github.com/bitmark-inc/bitmarkd/transaction"
	"testing"
)

// test to se iff state transition is allowed
func TestStateTransition(t *testing.T) {

	tests := []struct {
		old   transaction.State
		new   transaction.State
		valid bool
	}{
		{transaction.ExpiredTransaction, transaction.ExpiredTransaction, true},
		{transaction.ExpiredTransaction, transaction.UnpaidTransaction, true},
		{transaction.ExpiredTransaction, transaction.PendingTransaction, false},
		{transaction.ExpiredTransaction, transaction.ConfirmedTransaction, false},
		{transaction.ExpiredTransaction, transaction.MinedTransaction, false},
		{transaction.UnpaidTransaction, transaction.ExpiredTransaction, true},
		{transaction.UnpaidTransaction, transaction.UnpaidTransaction, true},
		{transaction.UnpaidTransaction, transaction.PendingTransaction, true},
		{transaction.UnpaidTransaction, transaction.ConfirmedTransaction, true},
		{transaction.UnpaidTransaction, transaction.MinedTransaction, true},
		{transaction.PendingTransaction, transaction.ExpiredTransaction, false},
		{transaction.PendingTransaction, transaction.UnpaidTransaction, false},
		{transaction.PendingTransaction, transaction.PendingTransaction, true},
		{transaction.PendingTransaction, transaction.ConfirmedTransaction, true},
		{transaction.PendingTransaction, transaction.MinedTransaction, true},
		{transaction.ConfirmedTransaction, transaction.ExpiredTransaction, false},
		{transaction.ConfirmedTransaction, transaction.UnpaidTransaction, false},
		{transaction.ConfirmedTransaction, transaction.PendingTransaction, false},
		{transaction.ConfirmedTransaction, transaction.ConfirmedTransaction, true},
		{transaction.ConfirmedTransaction, transaction.MinedTransaction, true},
		{transaction.MinedTransaction, transaction.ExpiredTransaction, false},
		{transaction.MinedTransaction, transaction.UnpaidTransaction, false},
		{transaction.MinedTransaction, transaction.PendingTransaction, false},
		{transaction.MinedTransaction, transaction.ConfirmedTransaction, false},
		{transaction.MinedTransaction, transaction.MinedTransaction, true},
	}

	for i, item := range tests {
		if item.old.CanChangeTo(item.new) != item.valid {
			t.Errorf("%d: transition from: %s  to: %s  is: %v  expected: %v", i, item.old, item.new, item.old.CanChangeTo(item.new), item.valid)
		}
	}

}
