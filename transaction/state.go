// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package transaction

import (
//"encoding/binary"
)

// type for transaction state
type State byte

// possible states for a transaction
const (
	ExpiredTransaction   = State('E')
	PendingTransaction   = State('P')
	VerifiedTransaction  = State('V')
	ConfirmedTransaction = State('C')
)

func (state State) CanChangeTo(newState State) bool {

	// exclude change to same state
	if state == newState {
		return false
	}

	switch state {
	case ExpiredTransaction:
		return PendingTransaction == newState

	case PendingTransaction:
		return true

	case VerifiedTransaction:
		return ConfirmedTransaction == newState

	default:
		return false
	}
}

func (state State) String() string {
	s := "?"
	switch state {
	case ExpiredTransaction:
		s = "Expired"
	case PendingTransaction:
		s = "Pending"
	case VerifiedTransaction:
		s = "Verified"
	case ConfirmedTransaction:
		s = "Confirmed"
	default:
	}
	return s
}

// convert a state to text for JSON
func (state State) MarshalJSON() ([]byte, error) {
	s, err := state.MarshalText()
	if nil != err {
		return nil, err
	}

	return []byte(`"` + string(s) + `"`), nil
}

// convert state to text
//
// Note: Each string _MUST_ start with a unique capital letter
// so client only need to test firrst character.
func (state State) MarshalText() ([]byte, error) {
	return []byte(state.String()), nil
}
