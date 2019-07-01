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
	actual := blockrecord.ValidBlockTimeSpacingAtVersion(1, 10)
	assert.Equal(t, nil, actual, "faile to validate valid v1 blcok time spacing")
}

func TestValidBlockTimeSpacingWhenCurrentVersionValid(t *testing.T) {
	actual := blockrecord.ValidBlockTimeSpacingAtVersion(2, uint64(100))
	assert.Equal(t, nil, actual, "faile to validate valid v2 blcok time spacing")
}

func TestValidBlockTimeSpacingWhenInitialVersionInvalid(t *testing.T) {
	actual := blockrecord.ValidBlockTimeSpacingAtVersion(1, uint64(100000))
	assert.Equal(t, fault.ErrInvalidBlockHeaderTimestamp, actual, "faile to validate invalid v1 blcok time spacing")
}

func TestValidBlockTimeSpacingWhenCurrentVersionInvalid(t *testing.T) {
	actual := blockrecord.ValidBlockTimeSpacingAtVersion(2, uint64(10000))
	assert.Equal(t, fault.ErrInvalidBlockHeaderTimestamp, actual, "faile to validate invalid v2 blcok time spacing")
}

func TestValidIncomingDifficutyWhenValid(t *testing.T) {
	difficulty.Current.Set(2)
	incoming := difficulty.New()
	incoming.Set(2)
	actual := blockrecord.ValidIncomingDifficuty(incoming)

	assert.Equal(t, nil, actual, "fail to validate valid incoming difficulty")
}

func TestValidIncomingDifficutyWhenInValid(t *testing.T) {
	difficulty.Current.Set(2)
	incoming := difficulty.New()
	incoming.Set(4)
	actual := blockrecord.ValidIncomingDifficuty(incoming)

	assert.Equal(t, fault.ErrDifficultyNotMatch, actual, "fail to validate invalid incoming difficulty")
}

func TestIsDifficultyAppliedVersionWhenApplied(t *testing.T) {
	actual := blockrecord.IsDifficultyAppliedVersion(3)
	assert.Equal(t, true, actual, "fail to check difficulty applied version")
}

func TestIsDifficultyAppliedVersionWhenNotApplied(t *testing.T) {
	actual := blockrecord.IsDifficultyAppliedVersion(1)
	assert.Equal(t, false, actual, "fail to check difficulty not applied version")
}

func TestValidHeaderVersionWhenTooSmall(t *testing.T) {
	actual := blockrecord.ValidHeaderVersion(uint16(10), uint16(0))
	assert.Equal(t, fault.ErrInvalidBlockHeaderVersion, actual, "fail to validate header version too small")
}

func TestValidHeaderVersionWhenPreviousLarger(t *testing.T) {
	actual := blockrecord.ValidHeaderVersion(uint16(100), uint16(99))
	assert.Equal(t, fault.ErrBlockVersionMustNotDecrease, actual, "fail to validate previous version larger")
}

func TestValidHeaderVersionWhenIncomingLarger(t *testing.T) {
	actual := blockrecord.ValidHeaderVersion(uint16(100), uint16(101))
	assert.Equal(t, nil, actual, "fail to validate incoming version larger")
}

func TestValidHeaderVersionWhenIncomingSame(t *testing.T) {
	actual := blockrecord.ValidHeaderVersion(uint16(100), uint16(100))
	assert.Equal(t, nil, actual, "fail to validate incoming version same")
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

	actual := blockrecord.ValidBlockLinkage(current, incoming)
	assert.Equal(t, fault.ErrPreviousBlockDigestDoesNotMatch, actual, "failt to validate different digest")
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

	actual := blockrecord.ValidBlockLinkage(current, incoming)
	assert.Equal(t, nil, actual, "failt to validate same digest")
}
