// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package util

import (
	"crypto/rand"
	"encoding/hex"

	crypto "github.com/libp2p/go-libp2p-core/crypto"
)

// MakeEd25519PeerKey generate a random ED25519 key in hex string format
func MakeEd25519PeerKey() (string, error) {
	r := rand.Reader
	privKey, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, 0, r)

	if err != nil {
		return "", err
	}

	return EncodePrivKeyToHex(privKey)
}

// DecodePrivKeyFromHex decode a hex string to a private key object
func DecodePrivKeyFromHex(privKey string) (crypto.PrivKey, error) {
	keyBytes, err := hex.DecodeString(privKey)
	if err != nil {
		return nil, err
	}

	key, err := crypto.UnmarshalPrivateKey(keyBytes)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// EncodePrivKeyToHex encode a private key object to a hex string
func EncodePrivKeyToHex(privKey crypto.PrivKey) (string, error) {
	keyBytes, err := crypto.MarshalPrivateKey(privKey)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(keyBytes), nil
}
