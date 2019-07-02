// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockrecord_test

import (
	"testing"

	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/stretchr/testify/assert"
)

func TestValidBlockTimeSpacingWhenInitialVersionValid(t *testing.T) {
	err := blockrecord.ValidBlockTimeSpacingAtVersion(1, 10)
	assert.Equal(t, nil, err, "valid initial block time spacing")
}

func TestValidBlockTimeSpacingWhenCurrentVersionValid(t *testing.T) {
	err := blockrecord.ValidBlockTimeSpacingAtVersion(2, uint64(100))
	assert.Equal(t, nil, err, "valid current block time spacing")
}

func TestValidBlockTimeSpacingWhenInitialVersionInvalid(t *testing.T) {
	err := blockrecord.ValidBlockTimeSpacingAtVersion(1, uint64(100000))
	assert.Equal(t, fault.ErrInvalidBlockHeaderTimestamp, err, "invalid initial block time spacing")
}

func TestValidBlockTimeSpacingWhenCurrentVersionInvalid(t *testing.T) {
	err := blockrecord.ValidBlockTimeSpacingAtVersion(2, uint64(10000))
	assert.Equal(t, fault.ErrInvalidBlockHeaderTimestamp, err, "invalid current block time spacing")
}

func TestValidIncomingDifficutyWhenValid(t *testing.T) {
	difficulty.Current.Set(2)
	incoming := difficulty.New()
	incoming.Set(2)
	err := blockrecord.ValidIncomingDifficuty(incoming)

	assert.Equal(t, nil, err, "valid incoming difficulty")
}

func TestValidIncomingDifficutyWhenInValid(t *testing.T) {
	difficulty.Current.Set(2)
	incoming := difficulty.New()
	incoming.Set(4)
	err := blockrecord.ValidIncomingDifficuty(incoming)

	assert.Equal(t, fault.ErrDifficultyNotMatch, err, "invalid incoming difficulty")
}

func TestIsDifficultyAppliedVersionWhenApplied(t *testing.T) {
	ok := blockrecord.IsDifficultyAppliedVersion(3)
	assert.Equal(t, true, ok, "difficulty applied version")
}

func TestIsDifficultyAppliedVersionWhenNotApplied(t *testing.T) {
	ok := blockrecord.IsDifficultyAppliedVersion(1)
	assert.Equal(t, false, ok, "difficulty not applied version")
}

func TestValidHeaderVersionWhenTooSmall(t *testing.T) {
	err := blockrecord.ValidHeaderVersion(uint16(10), uint16(0))
	assert.Equal(t, fault.ErrInvalidBlockHeaderVersion, err, "header version small")
}

func TestValidHeaderVersionWhenPreviousLarger(t *testing.T) {
	err := blockrecord.ValidHeaderVersion(uint16(100), uint16(99))
	assert.Equal(t, fault.ErrBlockVersionMustNotDecrease, err, "previous header version larger")
}

func TestValidHeaderVersionWhenIncomingLarger(t *testing.T) {
	err := blockrecord.ValidHeaderVersion(uint16(100), uint16(101))
	assert.Equal(t, nil, err, "incoming header version larger")
}

func TestValidHeaderVersionWhenIncomingSame(t *testing.T) {
	err := blockrecord.ValidHeaderVersion(uint16(100), uint16(100))
	assert.Equal(t, nil, err, "incoming header version same")
}

func TestValidBlockLinkageWhenInvalid(t *testing.T) {
	current := blockdigest.Digest{
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	}

	incoming := blockdigest.Digest{
		0x00, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	}

	err := blockrecord.ValidBlockLinkage(current, incoming)
	assert.Equal(t, fault.ErrPreviousBlockDigestDoesNotMatch, err, "incoming digest different")
}

func TestValidBlockLinkageWhenValid(t *testing.T) {
	current := blockdigest.Digest{
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	}

	incoming := blockdigest.Digest{
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	}

	err := blockrecord.ValidBlockLinkage(current, incoming)
	assert.Equal(t, nil, err, "incoming digest same")
}
