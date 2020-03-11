// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package announce_test

import (
	"testing"

	"github.com/bitmark-inc/bitmarkd/announce/fixtures"

	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/stretchr/testify/assert"
)

func TestInitialise(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()
	defer announce.Finalise()

	f := func(_ string) ([]string, error) { return []string{}, nil }

	err := announce.Initialise("domain.not.exist", "cache", f)
	assert.Nil(t, err, "wrong Initialise")
}

//func TestInitialiseWhenSecondTime(t *testing.T) {
//	fixtures.SetupTestLogger()
//	defer fixtures.TeardownTestLogger()
//	defer announce.Finalise()
//
//	f := func(_ string) ([]string, error) { return []string{}, nil }
//
//	_ = announce.Initialise("domain.not.exist", "cache", f)
//
//	err := announce.Initialise("domain.not.exist", "cache", f)
//
//	assert.Equal(t, fault.AlreadyInitialised, err, "wrong second Initialise")
//}

func TestFinalise(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	f := func(_ string) ([]string, error) { return []string{}, nil }

	err := announce.Initialise("domain.not.exist", "cache", f)
	assert.Nil(t, err, "wrong Initialise")

	err = announce.Finalise()
	assert.Nil(t, err, "wrong Finalise")
}
