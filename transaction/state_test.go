// Copyright (c) 2014-2016 Bitmark Inc.
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
		{transaction.ExpiredTransaction, transaction.ExpiredTransaction, false},
		{transaction.ExpiredTransaction, transaction.PendingTransaction, true},
		{transaction.ExpiredTransaction, transaction.VerifiedTransaction, false},
		{transaction.ExpiredTransaction, transaction.ConfirmedTransaction, false},

		{transaction.PendingTransaction, transaction.ExpiredTransaction, true},
		{transaction.PendingTransaction, transaction.PendingTransaction, false},
		{transaction.PendingTransaction, transaction.VerifiedTransaction, true},
		{transaction.PendingTransaction, transaction.ConfirmedTransaction, true},

		{transaction.VerifiedTransaction, transaction.ExpiredTransaction, false},
		{transaction.VerifiedTransaction, transaction.PendingTransaction, false},
		{transaction.VerifiedTransaction, transaction.VerifiedTransaction, false},
		{transaction.VerifiedTransaction, transaction.ConfirmedTransaction, true},

		{transaction.ConfirmedTransaction, transaction.ExpiredTransaction, false},
		{transaction.ConfirmedTransaction, transaction.PendingTransaction, false},
		{transaction.ConfirmedTransaction, transaction.VerifiedTransaction, false},
		{transaction.ConfirmedTransaction, transaction.ConfirmedTransaction, false},
	}

	for i, item := range tests {
		if item.old.CanChangeTo(item.new) != item.valid {
			t.Errorf("%d: transition from: %s  to: %s  is: %v  expected: %v", i, item.old, item.new, item.old.CanChangeTo(item.new), item.valid)
		}
	}

}
