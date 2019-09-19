package p2p

import (
	"crypto/rand"
	"encoding/hex"
	"errors"

	crypto "github.com/libp2p/go-libp2p-crypto"
)

// Identity of
type Identity struct {
	PrvKey crypto.PrivKey
}

// GenRandPrvKey Generate a random private key
func GenRandPrvKey() (crypto.PrivKey, error) {
	r := rand.Reader
	prvKey, _, err := crypto.GenerateEd25519Key(r)
	if err != nil {
		return nil, err
	}
	return prvKey, nil
}

// PublicKey get the public key of identity
func PublicKey(prvKey crypto.PrivKey) (crypto.PubKey, error) {
	if nil == prvKey {
		return nil, errors.New("private key is not initialized")
	}
	publicKey := prvKey.GetPublic()
	if nil == publicKey {
		return nil, errors.New("generate public key error")
	}
	return publicKey, nil
}

// EncodePrvKeyToHex  from hex encoded string to private key
func EncodePrvKeyToHex(prvKey crypto.PrivKey) ([]byte, error) {
	marshalKey, err := crypto.MarshalPrivateKey(prvKey)
	if err != nil {
		return nil, err
	}
	hexEncodeKey := make([]byte, hex.EncodedLen(len(marshalKey)))
	hex.Encode(hexEncodeKey, marshalKey)
	return hexEncodeKey, nil
}

//DecodeHexToPrvKey from hex encode to private key
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
