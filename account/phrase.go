package account

import (
	"github.com/bitmark-inc/bitmarkd/fault"
)

const (
	phraseV1Length = 24
	phraseV2Length = 12
)

var masks = []int{0, 1, 3, 7, 15, 31, 63, 127, 255, 511, 1023}

// Base58EncodedSeedToPhrase - convert base58 seed to recovery phrase
func Base58EncodedSeedToPhrase(encodedSeed string) ([]string, error) {

	sk, testnet, err := parseBase58Seed(encodedSeed)
	if nil != err {
		return nil, err
	}

	var phraseLength int

	switch len(sk) {
	case secretKeyV1Length:
		phraseLength = phraseV1Length

		// append network byte to sk
		if testnet {
			sk = append([]byte{0x01}, sk...)
		} else {
			sk = append([]byte{0x00}, sk...)
		}

	case secretKeyV2Length:
		phraseLength = phraseV2Length

	default:
		return nil, fault.ErrInvalidSecretKeyLength
	}

	phrase := make([]string, 0, phraseLength)
	accumulator := 0
	bits := 0
	n := 0

	for i := 0; i < len(sk); i++ {
		accumulator = accumulator<<8 + int(sk[i])
		bits += 8
		if bits >= 11 {
			bits -= 11
			n++
			index := accumulator >> uint(bits)
			accumulator &= masks[bits]
			word := bip39[index]
			phrase = append(phrase, word)
		}
	}

	if phraseLength != len(phrase) {
		return nil, fault.ErrInvalidPhraseLength
	}

	return phrase, nil
}
