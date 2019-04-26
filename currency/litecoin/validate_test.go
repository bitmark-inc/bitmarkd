// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package litecoin_test

import (
	"encoding/hex"
	"testing"

	"github.com/bitmark-inc/bitmarkd/currency/litecoin"
)

// for testing
type testAddress struct {
	address   string
	transform string
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
			transform: "MWvNfTBijRbimVRj8Nio26GJFBK4GxuUZn",
			version:   litecoin.LivenetScript,
			addrBytes: "fc85afab90ad569ed50fe8771d70aff8a7eb788d",
			valid:     true,
		},
		{
			address:   "MWvNfTBijRbimVRj8Nio26GJFBK4GxuUZn",
			transform: "3QiEMZmknJkHxz9q2VjTCT1tvUicLvBpdZ",
			version:   litecoin.LivenetScript2,
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
			transform: "QYsqNC1Ssu5veEyzZ7rFQExnqoKxHNARis",
			version:   litecoin.TestnetScript,
			addrBytes: "86a0ddc5ce64594f0b84d96596657e1f5e0af7f6",
			valid:     true,
		},
		{
			address:   "QYsqNC1Ssu5veEyzZ7rFQExnqoKxHNARis",
			transform: "2N5X5FB9Cro2qW4Dww1pEKYXMhQt8PK6KHM",
			version:   litecoin.TestnetScript2,
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
			transform: "M99pFUwmh7FnxRp9W9qF578XCXvE95NVs5",
			version:   litecoin.LivenetScript,
			addrBytes: "0dbdaf6928107d60299f5069367c4cf07fa9b6e5",
			valid:     true,
		},
		{
			address:   "M99pFUwmh7FnxRp9W9qF578XCXvE95NVs5",
			transform: "32wfwbXojzQN9vYFQGquFTt7sqKnB8Phyz",
			version:   litecoin.LivenetScript2,
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
			transform: "QfdNxV9u5w81ND7bqCdofCRgcpmDDqyzvH",
			version:   litecoin.TestnetScript,
			addrBytes: "d0ade0e231a81794ed1baa081604de53ddd8b083",
			valid:     true,
		},
		{
			address:   "QfdNxV9u5w81ND7bqCdofCRgcpmDDqyzvH",
			transform: "2NCGcqUHf4q4vE2MZD6bnaVzFUSKPM4WCDX",
			version:   litecoin.TestnetScript2,
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
			transform: "M8ptAjKsiTdNFGTkU5b6HiiroUV8cMiB87",
			version:   litecoin.LivenetScript,
			addrBytes: "0a290d74c272ab52dec1a87ce88e75d29c94fe5a",
			valid:     true,
		},
		{
			address:   "M8ptAjKsiTdNFGTkU5b6HiiroUV8cMiB87",
			transform: "32cjrquumLmwSmBrNCbkU5UTUmtgetWqaL",
			version:   litecoin.LivenetScript2,
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
			transform: "QbRxxXBkrmcARnustdYnvmjkiNofUE6yLm",
			version:   litecoin.TestnetScript,
			addrBytes: "a2a4c41bd7150d28aa730140cebf7aa5341e2619",
			valid:     true,
		},
		{
			address:   "QbRxxXBkrmcARnustdYnvmjkiNofUE6yLm",
			transform: "2N85CqWKWqfZ5Hc9qGXWmr5JKZzMqZCRDPM",
			version:   litecoin.TestnetScript2,
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
			transform: "MBQkZXHUBqPLs8BBqJ7bthVxFZSfwTYDn4",
			version:   litecoin.LivenetScript,
			addrBytes: "268118c8299cd5d8d3b9561caaf8c94d4bd1af44",
			valid:     true,
		},
		{
			address:   "MBQkZXHUBqPLs8BBqJ7bthVxFZSfwTYDn4",
			transform: "35CcFdsWEiXv4cuHjR8G54FYvrrDtm4WUm",
			version:   litecoin.LivenetScript2,
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
			transform: "QYR143nds6DEY92AmxD2R3Zne48UEHKAZB",
			version:   litecoin.TestnetScript,
			addrBytes: "818db8c869c5911d286d37088de9020cca43f702",
			valid:     true,
		},
		{
			address:   "QYR143nds6DEY92AmxD2R3Zne48UEHKAZB",
			transform: "2N54Ew2vPqzA9PxG89rB1LM8MVfgePiCKqV",
			version:   litecoin.TestnetScript2,
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
			transform: "MBPsxvnShawdBLk9GwH8aDoEgk5qKQwpf1",
			version:   litecoin.LivenetScript,
			addrBytes: "2656dc6ac50a5bdeb80348b9097af31e74698f44",
			valid:     true,
		},
		{
			address:   "MBPsxvnShawdBLk9GwH8aDoEgk5qKQwpf1",
			transform: "35Bjf3NUkU6CNqUFB4HnkaYqN3VPMYwUED",
			version:   litecoin.LivenetScript2,
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
			transform: "QRLAcQDU9FsMW9fZfmTxKP6y34oqQB6E7j",
			version:   litecoin.TestnetScript,
			addrBytes: "33dabd6dfda94c9c1ef1654a3c3b1e0984a7aecf",
			valid:     true,
		},
		{
			address:   "QRLAcQDU9FsMW9fZfmTxKP6y34oqQB6E7j",
			transform: "2MwyQVPME89pGMxuX3fRwEgfXtgN1Y1wB7e",
			version:   litecoin.TestnetScript2,
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
			transform: "MDuGE7b2sAbFeTbpctqB7jCxuYanknRkfs",
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
			transform: "QexMJG43LrBAutgVvP6ughSuAP8teN5ZBN",
			version:   litecoin.TestnetScript,
			addrBytes: "c94c4561b8ec99cddd540dedc67380c6b859ae00",
			valid:     true,
		},
		{
			address:   "QexMJG43LrBAutgVvP6ughSuAP8teN5ZBN",
			transform: "2NBbbBFBoKk85mhvTJH4tc11U1zh4oqp7SG",
			version:   litecoin.TestnetScript2,
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
			transform: "MHKKrwzqf8gosEgftwU5hLRDPGV8kPKrgj",
			version:   litecoin.LivenetScript,
			addrBytes: "674b274f2be5747793d2529972d964f1cfe9f985",
			valid:     true,
		},
		{
			address:   "MHKKrwzqf8gosEgftwU5hLRDPGV8kPKrgj",
			transform: "3B7BZ4asi1qP4jQmo4UjshAp4ZtgmpA7CW",
			version:   litecoin.LivenetScript2,
			addrBytes: "674b274f2be5747793d2529972d964f1cfe9f985",
			valid:     true,
		},

		{ // OLD:
			address: "2N2CSQaHQRzunj5SM9ZBwAPfJ3t85FMSze3A",
			//version:   litecoin.TestnetScript,
			//addrBytes: "2c7e3f628cd8c2028a7865e01ab8656f61da80913727ab467729",
			//valid:     true,
		},
		{ // NEW:
			address:   "QVZCXJGg21qpDd7CBHyBUMjV2WX5BJgkT6",
			transform: "2N2CSQHQRzunj5SM9ZBwAPfJ3t85FMSze3A",
			version:   litecoin.TestnetScript2,
			addrBytes: "62323f26365818cd158b2f6edde30148f9077a66",
			valid:     true,
		},
		{ // NEW:
			address:   "2N2CSQHQRzunj5SM9ZBwAPfJ3t85FMSze3A",
			transform: "QVZCXJGg21qpDd7CBHyBUMjV2WX5BJgkT6",
			version:   litecoin.TestnetScript,
			addrBytes: "62323f26365818cd158b2f6edde30148f9077a66",
			valid:     true,
		},
		{
			address:   "Qb7NQ3PjhvVBLJYTzTc834txdokHmHfLiS",
			transform: "2N7kcH2XVgpS6C7nRNMa6xNTXVRJTphp6Jt",
			version:   litecoin.TestnetScript2,
			addrBytes: "9f206df4a0fb27ff6614309c26e46ae9457a030e",
			valid:     true,
		},
		{
			address:   "2N7kcH2XVgpS6C7nRNMa6xNTXVRJTphp6Jt",
			transform: "Qb7NQ3PjhvVBLJYTzTc834txdokHmHfLiS",
			version:   litecoin.TestnetScript,
			addrBytes: "9f206df4a0fb27ff6614309c26e46ae9457a030e",
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

	for i, item := range addresses {
		actualVersion, actualBytes, err := litecoin.ValidateAddress(item.address)
		if item.valid {
			err := err
			if nil != err {
				t.Fatalf("%d: %s  error: %s", i, item.address, err)
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
				t.Fatalf("%d: version mismatch on: %s  actual: %d  expected: %d", i, item.address, actualVersion, int(item.version))
			}
			if actualBytes != expectedBytes {
				t.Fatalf("%d: bytes mismatch on: %s  actual: %x  expected: %x", i, item.address, actualBytes, expectedBytes)
			}

			// see if address transforms to its opposite type
			ta, err := litecoin.TransformAddress(item.address)
			if nil != err {
				t.Fatalf("%d: transfor error: %s", i, err)
			}
			transform := item.transform
			if "" == transform {
				transform = item.address
			}
			if ta != transform {
				t.Fatalf("%d: %s transformed to: %s  expected: %s", i, item.address, ta, transform)
			}
			//t.Logf("%d: version: %d bytes: %x", i, actualVersion, actualBytes)
		} else if nil == err {
			t.Errorf("%d: unexpected success", i)
		}
	}
}
