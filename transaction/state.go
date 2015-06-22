// Copyright (c) 2014-2015 Bitmark Inc.
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
	ExpiredTransaction      = State('E')
	UnpaidTransaction       = State('U')
	PendingTransaction      = State('P')
	ConfirmedTransaction    = State('C')
	MinedTransaction        = State('M')
)

func (state State) CanChangeTo(newState State) bool {
	if state == newState {
		return true
	}

	switch state {
	case ExpiredTransaction:
		return UnpaidTransaction == newState

	case UnpaidTransaction:
		return true

	case PendingTransaction:
		return ConfirmedTransaction == newState || MinedTransaction == newState
	case ConfirmedTransaction:
		return MinedTransaction == newState

	default:
		return false
	}
}

func (state State) String() string {
	s := "?"
	switch state {
	case ExpiredTransaction:
		s = "Expired"
	case UnpaidTransaction:
		s = "Unpaid"
	case PendingTransaction:
		s = "Pending"
	case ConfirmedTransaction:
		s = "Confirmed"
	case MinedTransaction:
		s = "Mined"
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
