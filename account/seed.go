// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package account

import (
	"bytes"
	"crypto/rand"
	"fmt"

	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/nacl/secretbox"
	"golang.org/x/crypto/sha3"

	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
)

// seed parameters
var (
	seedHeader   = []byte{0x5a, 0xfe}
	seedHeaderV1 = append(seedHeader, []byte{0x01}...)
	seedHeaderV2 = append(seedHeader, []byte{0x02}...)
)

// for seed v1 only
var (
	seedNonce = [24]byte{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	authSeedIndex = [16]byte{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03, 0xe7,
	}
)

const (
	seedHeaderLength = 3
	seedPrefixLength = 1

	secretKeyV1Length        = 32
	secretKeyV2Length        = 17
	secretKeyV2EntropyLength = 16
	seedChecksumLength       = 4

	seedV1Length = 40
	seedV2Length = 24
)

// PrivateKeyFromBase58Seed - this converts a Base58 encoded seed string and returns a private key
//
// one of the specific private key types are returned using the base "PrivateKeyInterface"
// interface type to allow individual methods to be called.
func PrivateKeyFromBase58Seed(seedBase58Encoded string) (*PrivateKey, error) {

	// verify length
	seed := util.FromBase58(seedBase58Encoded)
	seedLength := len(seed)
	if seedV1Length != seedLength && seedV2Length != seedLength {
		return nil, fault.ErrInvalidSeedLength
	}

	// verify checksum
	digest := sha3.Sum256(seed[:seedLength-checksumLength])
	checksumStart := seedLength - seedChecksumLength
	expectedChecksum := digest[:seedChecksumLength]
	actualChecksum := seed[checksumStart:]
	if !bytes.Equal(expectedChecksum, actualChecksum) {
		return nil, fault.ErrChecksumMismatch
	}

	header := seed[:seedHeaderLength]
	var ed25519Seed []byte // the ed25519 seed for generate key pair
	var testnet bool       // denote the network is test net

	switch {
	case bytes.Equal(seedHeaderV1, header):
		// copy the secret key from seed
		var sk [secretKeyV1Length]byte
		secretStart := seedHeaderLength + seedPrefixLength
		copy(sk[:], seed[secretStart:])

		prefix := seed[seedHeaderLength:secretStart]
		// first byte of prefix is test/live indication
		testnet = prefix[0] == 0x01

		ed25519Seed = secretbox.Seal([]byte{}, authSeedIndex[:], &seedNonce, &sk)

	case bytes.Equal(seedHeaderV2, header):
		sk := seed[seedHeaderLength:checksumStart]

		// verify valid secret key
		if secretKeyV2Length != len(sk) || 0 != sk[16]&0x0f {
			return nil, fault.ErrInvalidSeedLength
		}

		// parse network
		mode := sk[0]&0x80 | sk[1]&0x40 | sk[2]&0x20 | sk[3]&0x10
		testnet = mode == sk[15]&0xf0^0xf0

		// add the seed 4 times to hash value
		hash := sha3.NewShake256()
		for i := 0; i < 4; i++ {
			n, err := hash.Write(sk)
			if err != nil {
				return nil, err
			}
			if secretKeyV2Length != n {
				return nil, fault.ErrCannotDecodeSeed
			}
		}

		ed25519Seed = make([]byte, ed25519.SeedSize)
		n, err := hash.Read(ed25519Seed)
		if nil != err {
			return nil, err
		}
		if ed25519.SeedSize != n {
			return nil, fault.ErrCannotDecodeSeed
		}

	default:
		return nil, fault.ErrInvalidSeedHeader
	}

	// generate key pair from encrypted secret key
	_, priv, err := ed25519.GenerateKey(bytes.NewBuffer(ed25519Seed))
	if nil != err {
		return nil, err
	}

	privateKey := &PrivateKey{
		PrivateKeyInterface: &ED25519PrivateKey{
			Test:       testnet,
			PrivateKey: priv,
		},
	}
	return privateKey, nil
}

// NewBase58EncodedSeedV1 - generate base58 seed v1
func NewBase58EncodedSeedV1(testnet bool) (string, error) {
	// generate new seed
	sk := make([]byte, secretKeyV1Length)
	n, err := rand.Read(sk)
	if nil != err {
		return "", err
	}
	if secretKeyV1Length != n {
		return "", fmt.Errorf("got: %d bytes, expected: %d bytes", n, secretKeyV1Length)
	}
	net := 0x00
	if testnet {
		net = 0x01
	}
	seed := append(seedHeaderV1, byte(net))
	seed = append(seed, sk...)
	checksum := sha3.Sum256(seed)
	seed = append(seed, checksum[:seedChecksumLength]...)

	base58Encodedseed := util.ToBase58(seed)
	return base58Encodedseed, nil
}

// NewBase58EncodedSeedV2 - generate base58 seed v2
func NewBase58EncodedSeedV2(testnet bool) (string, error) {

	// space for 128 bit, extend to 132 bit later
	sk := make([]byte, secretKeyV2EntropyLength, secretKeyV2Length)

	n, err := rand.Read(sk)
	if nil != err {
		return "", err
	}

	if secretKeyV2EntropyLength != n {
		return "", fmt.Errorf("got: %d bytes, expected: %d bytes", n, secretKeyV2EntropyLength)
	}

	// extend to 132 bits
	sk = append(sk, sk[15]&0xf0)

	if secretKeyV2Length != len(sk) {
		return "", fmt.Errorf("actual seed length is %d bytes, expected is %d bytes", len(sk), secretKeyV2Length)
	}

	// network flag
	mode := sk[0]&0x80 | sk[1]&0x40 | sk[2]&0x20 | sk[3]&0x10
	if testnet {
		mode = mode ^ 0xf0
	}
	sk[15] = mode | sk[15]&0x0f

	// encode seed to base58
	seed := append(seedHeaderV2, sk...)
	digest := sha3.Sum256(seed)
	checksum := digest[:seedChecksumLength]
	seed = append(seed, checksum...)

	base58EncodedSeed := util.ToBase58(seed)

	return base58EncodedSeed, nil
}
