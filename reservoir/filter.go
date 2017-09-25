package reservoir

import (
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/sha3"
)

var (
	ErrInvalidLength = fmt.Errorf("invalid length of proof filter")
)

// ProofFilter is a bloom filter which is used to examine whether an issue is verified or not
type ProofFilter [4096]byte

// Add a transaction to the proof filter
func (p *ProofFilter) Add(transaction []byte) {
	filter := p[:]
	hash := sha3.Sum384(transaction)
	for i := 0; i < 28; i++ {
		var n uint
		if i%2 == 0 {
			n = uint(hash[3*i/2])<<4 | uint(hash[3*i/2+1]>>4)
		} else {
			n = uint(hash[3*(i-1)/2+1]&15)<<8 | uint(hash[3*(i-1)/2+2])
		}
		filter[n] = 1
	}
}

// Has is to check whether a transaction is added into the filter
func (p ProofFilter) Has(transaction []byte) bool {
	hash := sha3.Sum384(transaction)
	for i := 0; i < 28; i++ {
		var n uint
		if i%2 == 0 {
			n = uint(hash[3*i/2])<<4 | uint(hash[3*i/2+1]>>4)
		} else {
			n = uint(hash[3*(i-1)/2+1]&15)<<8 | uint(hash[3*(i-1)/2+2])
		}

		if v := p[n]; v != 1 {
			return false
		}
	}
	return true
}

// MarshalText is for encode the filter to JSON
func (p ProofFilter) MarshalText() ([]byte, error) {
	b := make([]byte, 8192)
	hex.Encode(b, p[:])
	return b, nil
}

// UnmarshalText is for decode the filter from JSON
func (p *ProofFilter) UnmarshalText(s []byte) error {
	b := make([]byte, 4096)
	_, err := hex.Decode(b, s)
	if err != nil {
		return err
	}
	if len(b) != 4096 {
		return ErrInvalidLength
	}

	copy(p[:], b[:])
	return nil
}
