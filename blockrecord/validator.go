// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockrecord

import (
	"github.com/bitmark-inc/bitmarkd/difficulty"
)

const (
	blockTimeSpacingInitial = 240 * 60
	blockTimeSpacingCurrent = 10 * 60

	initiailVersion              = 1
	modififiedTimeSpacingVersion = 2
	difficultyAppliedVersion     = 3
)

// ValidBlockTimeSpacingAtVersion - valid block time spacing based on different version
func ValidBlockTimeSpacingAtVersion(version uint16, timeSpacing uint64) bool {
	if version == initiailVersion {
		return timeSpacing <= blockTimeSpacingInitial
	}

	if version >= modififiedTimeSpacingVersion {
		return timeSpacing <= blockTimeSpacingCurrent
	}

	return false
}

// ValidIncomingDifficuty - valid incoming difficulty
func ValidIncomingDifficuty(incoming *difficulty.Difficulty) bool {
	return incoming.Value() == difficulty.Current.Value()
}

// ValidDifficultyAppliedVersion - is difficulty rule applied at header version
func ValidDifficultyAppliedVersion(version uint16) bool {
	return version >= difficultyAppliedVersion
}
