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
	WaitingIssueTransaction = State('W')
	UnpaidTransaction       = State('U')
	AvailableTransaction    = State('A')
	MinedTransaction        = State('M')
)

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
	s := "?"
	switch state {
	case ExpiredTransaction:
		s = "Expired"
	case WaitingIssueTransaction:
		s = "Waiting"
	case UnpaidTransaction:
		s = "Unpaid"
	case AvailableTransaction:
		s = "Available"
	case MinedTransaction:
		s = "Mined"
	default:
	}
	return []byte(s), nil
}
