// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package litecoin_test

import (
	"encoding/hex"
	"github.com/bitmark-inc/bitmarkd/currency/litecoin"
	"testing"
)

// for testing
type testAddress struct {
	address   string
	version   litecoin.Version
	addrBytes string
	valid     bool
}

func TestValidate(t *testing.T) {

	addresses := []testAddress{
		{
			address:   "LdwLvykqj2nUH3MWcut6mtjHxVxVFC7st5",
			version:   litecoin.Livenet,
			addrBytes: "cd463dbc6f8076c7021f2766b36ea7e19c5c9e2e",
			valid:     true,
		},
		{
			address:   "3QiEMZmknJkHxz9q2VjTCT1tvUicLvBpdZ",
			version:   litecoin.LivenetScript,
			addrBytes: "fc85afab90ad569ed50fe8771d70aff8a7eb788d",
			valid:     true,
		},
		{
			address:   "mmCKZS7toE69QgXNs1JZcjW6LFj8LfUbz6",
			version:   litecoin.Testnet,
			addrBytes: "3e4a9a4a79dcad8800b6cfcdf102bf98064b7454",
			valid:     true,
		},
		{
			address:   "2N5X5FB9Cro2qW4Dww1pEKYXMhQt8PK6KHM",
			version:   litecoin.TestnetScript,
			addrBytes: "86a0ddc5ce64594f0b84d96596657e1f5e0af7f6",
			valid:     true,
		},
		{
			address:   "LWZR9ybwmT8vSXP6tmrBX4b6nE9o94AjQG",
			version:   litecoin.Livenet,
			addrBytes: "7c57bc50a38d8377ad55260f29f2c8619846ef08",
			valid:     true,
		},
		{
			address:   "32wfwbXojzQN9vYFQGquFTt7sqKnB8Phyz",
			version:   litecoin.LivenetScript,
			addrBytes: "0dbdaf6928107d60299f5069367c4cf07fa9b6e5",
			valid:     true,
		},
		{
			address:   "mvJg85FLYqN7xAcZeFZRVg7pMbJ53BqKmy",
			version:   litecoin.Testnet,
			addrBytes: "a237653c5ae7e18e840d6463d380701ce3ba5035",
			valid:     true,
		},
		{
			address:   "2NCGcqUHf4q4vE2MZD6bnaVzFUSKPM4WCDX",
			version:   litecoin.TestnetScript,
			addrBytes: "d0ade0e231a81794ed1baa081604de53ddd8b083",
			valid:     true,
		},
		{
			address:   "LWkdEB9SHUfuBiTvZofK2LqYE4RTTtUcqi",
			version:   litecoin.Livenet,
			addrBytes: "7e766382cb564021bcbc273e23569dcaed536ac6",
			valid:     true,
		},
		{
			address:   "32cjrquumLmwSmBrNCbkU5UTUmtgetWqaL",
			version:   litecoin.LivenetScript,
			addrBytes: "0a290d74c272ab52dec1a87ce88e75d29c94fe5a",
			valid:     true,
		},
		{
			address:   "mtei3esVvHhww4Rw9FYnMdTUTVvbpWhLfF",
			version:   litecoin.Testnet,
			addrBytes: "901111ab28cf850a5b6846e94e8c0c4a505603a9",
			valid:     true,
		},
		{
			address:   "2N85CqWKWqfZ5Hc9qGXWmr5JKZzMqZCRDPM",
			version:   litecoin.TestnetScript,
			addrBytes: "a2a4c41bd7150d28aa730140cebf7aa5341e2619",
			valid:     true,
		},
		{
			address:   "LVcGHJcTv1ctR6GLRXxR4SQSsycdmQ6pwZ",
			version:   litecoin.Livenet,
			addrBytes: "71e9734a1283f2368bbd5a397d3c7a22610b2958",
			valid:     true,
		},
		{
			address:   "35CcFdsWEiXv4cuHjR8G54FYvrrDtm4WUm",
			version:   litecoin.LivenetScript,
			addrBytes: "268118c8299cd5d8d3b9561caaf8c94d4bd1af44",
			valid:     true,
		},
		{
			address:   "myWBvpVEeY86YvJLb5kwH2iWbdXPGjTtZk",
			version:   litecoin.Testnet,
			addrBytes: "c54d3aa920e78e56b72c0076d36e99bc87058397",
			valid:     true,
		},
		{
			address:   "2N54Ew2vPqzA9PxG89rB1LM8MVfgePiCKqV",
			version:   litecoin.TestnetScript,
			addrBytes: "818db8c869c5911d286d37088de9020cca43f702",
			valid:     true,
		},
		{
			address:   "LPD8ZwGjE4WmQ1EEnjZHrvofSyvGtbEWsH",
			version:   litecoin.Livenet,
			addrBytes: "2bb8b0991f396d7f411c2227af00cc09d1ae0adf",
			valid:     true,
		},
		{
			address:   "35Bjf3NUkU6CNqUFB4HnkaYqN3VPMYwUED",
			version:   litecoin.LivenetScript,
			addrBytes: "2656dc6ac50a5bdeb80348b9097af31e74698f44",
			valid:     true,
		},
		{
			address:   "mhv2Ti1xy9CsWoYgnEjehEunbhFiyFwLAp",
			version:   litecoin.Testnet,
			addrBytes: "1a4d4bf230aabafd3a425770b8b98700bf06e370",
			valid:     true,
		},
		{
			address:   "2MwyQVPME89pGMxuX3fRwEgfXtgN1Y1wB7e",
			version:   litecoin.TestnetScript,
			addrBytes: "33dabd6dfda94c9c1ef1654a3c3b1e0984a7aecf",
			valid:     true,
		},
		{
			address:   "LPGeGFBPCVLHdGVD1i1oikzD92XZoTEVyh",
			version:   litecoin.Livenet,
			addrBytes: "2c62b9d0c13b499167506863248f473416b18850",
			valid:     true,
		},
		{
			address:   "37h7vEB4v3jpqxKvX1qqJ5xZaqzLj7NPyN",
			version:   litecoin.LivenetScript,
			addrBytes: "41d5c23a8188270b32d0afce2e11e4c3028afe6b",
			valid:     true,
		},
		{
			address:   "mhvk8vH4LaAgUBUJsU4UtL4KSWLavssToW",
			version:   litecoin.Testnet,
			addrBytes: "1a701609b7d938f932d9517f965eb938ec45d067",
			valid:     true,
		},
		{
			address:   "2NBbbBFBoKk85mhvTJH4tc11U1zh4oqp7SG",
			version:   litecoin.TestnetScript,
			addrBytes: "c94c4561b8ec99cddd540dedc67380c6b859ae00",
			valid:     true,
		},
		{
			address:   "LhLu7S8qdG7YZR1GgSP8g4aqN8nXCRLkzX",
			version:   litecoin.Livenet,
			addrBytes: "f2a30c60e4abcbbdcdf7cb34520b742ae07b6018",
			valid:     true,
		},
		{
			address:   "3B7BZ4asi1qP4jQmo4UjshAp4ZtgmpA7CW",
			version:   litecoin.LivenetScript,
			addrBytes: "674b274f2be5747793d2529972d964f1cfe9f985",
			valid:     true,
		},
		{
			address: "LgcotVvFQgGHygDWCkkyqVgyctGTe3pH4G",
		},
		{
			address: "2PGCEn9m7Vx1hcwjZzvm61UBfdm8rynZcNb",
		},
		{
			address: "2iKcv8HMvVtbHVEmfPPp52AMbjLQqgvbFYt",
		},
	}

	/*

			{
				address:   "mipcBbFg9gMiCh81Kj8tqqdgoZub1ZJRfn",
				version:   litecoin.Testnet,
				addrBytes: "243f1394f44554f4ce3fd68649c19adc483ce924",
				valid:     true,
			},
			{
				address:   "2MzQwSSnBHWHqSAqtTVQ6v47XtaisrJa1Vc",
				version:   litecoin.TestnetScript,
				addrBytes: "4e9f39ca4688ff102128ea4ccda34105324305b0",
				valid:     true,
			},
			{
				address:   "17VZNX1SN5NtKa8UQFxwQbFeFc3iqRYhem",
				version:   litecoin.Livenet,
				addrBytes: "47376c6f537d62177a2c41c4ca9b45829ab99083",
				valid:     true,
			},
			{
				address:   "3EktnHQD7RiAE6uzMj2ZifT9YgRrkSgzQX",
				version:   litecoin.LivenetScript,
				addrBytes: "8f55563b9a19f321c211e9b9f38cdf686ea07845",
				valid:     true,
			},
			{
				address:   "3EktnHQD7RiAE6uzMj2ZifT9YgRrkSgzQZ",
				version:   litecoin.LivenetScript,
				addrBytes: "0000000000000000000000000000000000000000",
				valid:     false,
			},
			{
				address:   "mipcBbFg9gMiCh81Kj9tqqdgoZub1ZJRfn",
				version:   litecoin.Testnet,
				addrBytes: "0000000000000000000000000000000000000000",
				valid:     false,
			},
		}
	*/

	for i, item := range addresses {
		actualVersion, actualBytes, err := litecoin.ValidateAddress(item.address)
		if item.valid {
			if nil != err {
				t.Fatalf("%d: error: %s", i, err)
			}
			eb, err := hex.DecodeString(item.addrBytes)
			if nil != err {
				t.Fatalf("%d: hex decode error: %s", i, err)
			}
			expectedBytes := litecoin.AddressBytes{}
			if len(eb) != len(expectedBytes) {
				t.Fatalf("%d: hex length actual: %d expected: %d", i, len(eb), len(expectedBytes))
			}
			copy(expectedBytes[:], eb)

			if actualVersion != item.version {
				t.Errorf("%d: version mismatch actual: %d expected: %d", i, actualVersion, item.version)
			}
			if actualBytes != expectedBytes {
				t.Errorf("%d: bytes mismatch actual: %x expected: %x", i, actualBytes, expectedBytes)
			}

			t.Logf("%d: version: %d bytes: %x", i, actualVersion, actualBytes)
		} else if nil == err {
			t.Errorf("%d: unexpected success", i)
		}
	}
}
