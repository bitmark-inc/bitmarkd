// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc_test

import (
	"fmt"
	"os"

	"github.com/bitmark-inc/logger"
)

const (
	testingDirName  = "testing"
	logCategory     = "testing"
	bitcoinAddress  = "2N7uK4otZGYDUDNEQ3Yr6hPPrs49BHQA32L"
	litecoinAddress = "mwLH3WTj4zxMSM3Tzq3w9rfgJicawtKp1R"
)

var (
	issuerPublicKey = []byte{
		0x7a, 0x81, 0x92, 0x56, 0x5e, 0x6c, 0xa2, 0x35,
		0x80, 0xe1, 0x81, 0x59, 0xef, 0x30, 0x73, 0xf6,
		0xe2, 0xfb, 0x8e, 0x7e, 0x9d, 0x31, 0x49, 0x7e,
		0x79, 0xd7, 0x73, 0x1b, 0xa3, 0x74, 0x11, 0x01,
	}
	issuerPrivateKey = []byte{
		0x66, 0xf5, 0x28, 0xd0, 0x2a, 0x64, 0x97, 0x3a,
		0x2d, 0xa6, 0x5d, 0xb0, 0x53, 0xea, 0xd0, 0xfd,
		0x94, 0xca, 0x93, 0xeb, 0x9f, 0x74, 0x02, 0x3e,
		0xbe, 0xdb, 0x2e, 0x57, 0xb2, 0x79, 0xfd, 0xf3,
		0x7a, 0x81, 0x92, 0x56, 0x5e, 0x6c, 0xa2, 0x35,
		0x80, 0xe1, 0x81, 0x59, 0xef, 0x30, 0x73, 0xf6,
		0xe2, 0xfb, 0x8e, 0x7e, 0x9d, 0x31, 0x49, 0x7e,
		0x79, 0xd7, 0x73, 0x1b, 0xa3, 0x74, 0x11, 0x01,
	}
	receiverPublicKey = []byte{
		0x9f, 0xc4, 0x86, 0xa2, 0x53, 0x4f, 0x17, 0xe3,
		0x67, 0x07, 0xfa, 0x4b, 0x95, 0x3e, 0x3b, 0x34,
		0x00, 0xe2, 0x72, 0x9f, 0x65, 0x61, 0x16, 0xdd,
		0x7b, 0x01, 0x8d, 0xf3, 0x46, 0x98, 0xbd, 0xc2,
	}
	receiverPrivateKey = []byte{
		0xf3, 0xf7, 0xa1, 0xfc, 0x33, 0x10, 0x71, 0xc2,
		0xb1, 0xcb, 0xbe, 0x4f, 0x3a, 0xee, 0x23, 0x5a,
		0xae, 0xcc, 0xd8, 0x5d, 0x2a, 0x80, 0x4c, 0x44,
		0xb5, 0xc6, 0x03, 0xb4, 0xca, 0x4d, 0x9e, 0xc0,
		0x9f, 0xc4, 0x86, 0xa2, 0x53, 0x4f, 0x17, 0xe3,
		0x67, 0x07, 0xfa, 0x4b, 0x95, 0x3e, 0x3b, 0x34,
		0x00, 0xe2, 0x72, 0x9f, 0x65, 0x61, 0x16, 0xdd,
		0x7b, 0x01, 0x8d, 0xf3, 0x46, 0x98, 0xbd, 0xc2,
	}
)

func setupTestLogger() {
	removeFiles()
	_ = os.Mkdir(testingDirName, 0700)

	logging := logger.Configuration{
		Directory: testingDirName,
		File:      fmt.Sprintf("%s.Log", logCategory),
		Size:      1048576,
		Count:     10,
		Console:   false,
		Levels: map[string]string{
			logger.DefaultTag: "critical",
		},
	}

	// Start logging
	_ = logger.Initialise(logging)
}

func teardownTestLogger() {
	removeFiles()
}

func removeFiles() {
	_ = os.RemoveAll(testingDirName)
}
