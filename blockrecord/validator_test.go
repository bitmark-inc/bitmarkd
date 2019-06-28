// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockrecord_test

import (
	"testing"

	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/stretchr/testify/assert"
)

func TestValidBlockTimeSpacingWhenV1Valid(t *testing.T) {
	actual := blockrecord.ValidBlockTimeSpacingAtVersion(1, 10)
	assert.Equal(t, true, actual, "faile to validate valid v1 blcok time spacing")
}

func TestValidBlockTimeSpacingWhenV1InValid(t *testing.T) {
	actual := blockrecord.ValidBlockTimeSpacingAtVersion(1, uint64(100000))
	assert.Equal(t, false, actual, "faile to validate invalid v1 blcok time spacing")
}

func TestValidBlockTimeSpacingWhenV2Valid(t *testing.T) {
	actual := blockrecord.ValidBlockTimeSpacingAtVersion(2, uint64(100))
	assert.Equal(t, true, actual, "faile to validate valid v2 blcok time spacing")
}

func TestValidBlockTimeSpacingWhenV2InValid(t *testing.T) {
	actual := blockrecord.ValidBlockTimeSpacingAtVersion(2, uint64(10000))
	assert.Equal(t, false, actual, "faile to validate invalid v2 blcok time spacing")
}

func TestValidIncomingDifficutyWhenValid(t *testing.T) {
	difficulty.Current.Set(2)
	incoming := difficulty.New()
	incoming.Set(2)
	actual := blockrecord.ValidIncomingDifficuty(incoming)

	assert.Equal(t, true, actual, "fail to validate valid incoming difficulty")
}

func TestValidIncomingDifficutyWhenInValid(t *testing.T) {
	difficulty.Current.Set(2)
	incoming := difficulty.New()
	incoming.Set(4)
	actual := blockrecord.ValidIncomingDifficuty(incoming)

	assert.Equal(t, false, actual, "fail to validate invalid incoming difficulty")
}

func TestValidDifficultyAppliedVersionWhenApplied(t *testing.T) {
	actual := blockrecord.ValidDifficultyAppliedVersion(3)
	assert.Equal(t, true, actual, "fail to check difficulty applied version")
}

func TestValidDifficultyAppliedVersionWhenNotApplied(t *testing.T) {
	actual := blockrecord.ValidDifficultyAppliedVersion(1)
	assert.Equal(t, false, actual, "fail to check difficulty not applied version")
}
