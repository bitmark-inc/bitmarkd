package fingerprint

import (
	"encoding"
	"encoding/hex"
)

type Type [32]byte

type Fingerprint interface {
	encoding.TextMarshaler
}

// MarshalText - encode as hex
func (t Type) MarshalText() ([]byte, error) {
	buffer := make([]byte, hex.EncodedLen(len(t)))
	hex.Encode(buffer, t[:])

	return buffer, nil
}
