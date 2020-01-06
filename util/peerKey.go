package util

import (
	"crypto/rand"
	"encoding/hex"

	crypto "github.com/libp2p/go-libp2p-core/crypto"
)

//GenEd25519Key Generate a random ED29919
func MakeEd25519PeerKey() (string, error) {
	r := rand.Reader
	prvKey, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, 0, r)

	if err != nil {
		return "", err
	}

	keyBytes, err := crypto.MarshalPrivateKey(prvKey)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(keyBytes), nil
}

//DecodeHexToPrvKey decode a hex encoded key to a PrivKey
func DecodeHexToPrvKey(prvKey []byte) (crypto.PrivKey, error) {
	hexDecodeKey := make([]byte, hex.DecodedLen(len(prvKey)))
	_, err := hex.Decode(hexDecodeKey, prvKey)
	if err != nil {
		return nil, err
	}
	unmarshalKey, err := crypto.UnmarshalPrivateKey(hexDecodeKey)
	if err != nil {
		return nil, err
	}
	return unmarshalKey, nil
}

// EncodePrvKeyToHex  HexEncode a PrivKey
func EncodePrvKeyToHex(prvKey crypto.PrivKey) ([]byte, error) {
	marshalKey, err := crypto.MarshalPrivateKey(prvKey)
	if err != nil {
		return nil, err
	}
	hexEncodeKey := make([]byte, hex.EncodedLen(len(marshalKey)))
	hex.Encode(hexEncodeKey, marshalKey)
	return hexEncodeKey, nil
}
