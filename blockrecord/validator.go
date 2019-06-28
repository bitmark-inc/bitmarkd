// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockrecord

import (
	"github.com/bitmark-inc/bitmarkd/difficulty"
)

const (
	blockTimeSpacingV1 = 240 * 60
	blockTimeSpacingV2 = 10 * 60
)

// ValidBlockTimeSpacingAtVersion - valid block time spacing based on different version
func ValidBlockTimeSpacingAtVersion(version uint16, timeSpacing uint64) bool {
	if version == 1 {
		return timeSpacing <= blockTimeSpacingV1
	}

	if version >= 2 {
		return timeSpacing <= blockTimeSpacingV2
	}

	return false
}

// ValidIncomingDifficuty - valid incoming difficulty
func ValidIncomingDifficuty(incoming *difficulty.Difficulty) bool {
	return incoming.Value() == difficulty.Current.Value()
}

// ValidDifficultyAppliedVersion - is difficulty rule applied at header version
func ValidDifficultyAppliedVersion(version uint16) bool {
	return version >= 3
}
