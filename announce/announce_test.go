// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package announce_test

import (
	"testing"
	"time"

	"github.com/bitmark-inc/bitmarkd/fault"

	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/announce/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestInitialise(t *testing.T) {
	_ = announce.Finalise()

	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()
	defer announce.Finalise()

	f := func(_ string) ([]string, error) { return []string{}, nil }

	err := announce.Initialise("domain.not.exist", "cache", f)

	// make sure background jobs already finish first round, so
	// no logger will be called
	time.Sleep(10 * time.Millisecond)

	assert.Nil(t, err, "wrong Initialise")
}

func TestInitialiseWhenSecondTime(t *testing.T) {
	_ = announce.Finalise()

	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()
	defer announce.Finalise()

	f := func(_ string) ([]string, error) { return []string{}, nil }

	_ = announce.Initialise("domain.not.exist", "cache", f)

	err := announce.Initialise("domain.not.exist", "cache", f)

	// make sure background jobs already finish first round, so
	// no logger will be called
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, fault.AlreadyInitialised, err, "wrong second Initialise")
}

func TestFinalise(t *testing.T) {
	_ = announce.Finalise()

	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	f := func(_ string) ([]string, error) { return []string{}, nil }

	err := announce.Initialise("domain.not.exist", "cache", f)
	assert.Nil(t, err, "wrong Initialise")

	// make sure background jobs already finish first round, so
	// no logger will be called
	time.Sleep(10 * time.Millisecond)

	err = announce.Finalise()
	assert.Nil(t, err, "wrong Finalise")
}
