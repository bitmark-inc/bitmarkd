// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package certificate_test

import (
	"crypto/tls"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/sha3"

	"github.com/bitmark-inc/bitmarkd/rpc/certificate"
	"github.com/bitmark-inc/bitmarkd/rpc/fixtures"
	"github.com/bitmark-inc/logger"
)

func TestGet(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	wd, _ := os.Getwd()
	fixtureDir := path.Join(filepath.Dir(wd), "fixtures")
	cer := fixtures.Certificate(fixtureDir)
	key := fixtures.Key(fixtureDir)

	tlsConfig, fingerprint, err := certificate.Get(
		logger.New(fixtures.LogCategory),
		"test",
		cer,
		key,
	)
	assert.Nil(t, err, "wrong Get")

	pair, _ := tls.X509KeyPair([]byte(cer), []byte(key))

	assert.Equal(t, sha3.Sum256(pair.Certificate[0]), fingerprint, "wrong fingerprint")
	assert.Equal(t, pair, tlsConfig.Certificates[0], "wrong config")
}
