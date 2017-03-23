// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package reservoir

// result of track payment/try proof
type TrackingStatus int

// possible status values
const (
	TrackingNotFound  TrackingStatus = iota
	TrackingAccepted  TrackingStatus = iota
	TrackingProcessed TrackingStatus = iota
	TrackingInvalid   TrackingStatus = iota
)

// convert the tracking value for printf
func (ts TrackingStatus) String() string {
	switch ts {
	case TrackingNotFound:
		return "NotFound"
	case TrackingAccepted:
		return "Accepted"
	case TrackingProcessed:
		return "Processed"
	case TrackingInvalid:
		return "Invalid"
	default:
		return "*Unknown*"
	}
}

// convert the tracking value for JSON
func (ts TrackingStatus) MarshalText() ([]byte, error) {
	buffer := []byte(ts.String())
	return buffer, nil
}

// convert the tracking value from JSON to enumeration
func (ts *TrackingStatus) UnmarshalText(s []byte) error {
	switch string(s) {
	case "NotFound":
		*ts = TrackingNotFound
	case "Accepted":
		*ts = TrackingAccepted
	case "Processed":
		*ts = TrackingProcessed
	case "Invalid":
		*ts = TrackingInvalid
	default:
		*ts = TrackingInvalid
	}
	return nil
}
