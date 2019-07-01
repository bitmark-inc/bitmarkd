// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockrecord

import (
	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/bitmark-inc/bitmarkd/fault"
)

const (
	blockTimeSpacingInitial = 240 * 60
	blockTimeSpacingCurrent = 10 * 60

	initialVersion               = 1
	modififiedTimeSpacingVersion = 2
	difficultyAppliedVersion     = 3
)

// ValidBlockTimeSpacingAtVersion - valid block time spacing based on different version
func ValidBlockTimeSpacingAtVersion(version uint16, timeSpacing uint64) error {
	if version == initialVersion && timeSpacing > blockTimeSpacingInitial {
		return fault.ErrInvalidBlockHeaderTimestamp
	}

	if version >= modififiedTimeSpacingVersion && timeSpacing > blockTimeSpacingCurrent {
		return fault.ErrInvalidBlockHeaderTimestamp
	}

	return nil
}

// ValidIncomingDifficuty - valid incoming difficulty
func ValidIncomingDifficuty(incoming *difficulty.Difficulty) error {
	if incoming.Value() != difficulty.Current.Value() {
		return fault.ErrDifficultyNotMatch
	}
	return nil
}

// IsDifficultyAppliedVersion - is difficulty rule applied at header version
func IsDifficultyAppliedVersion(version uint16) bool {
	return version >= difficultyAppliedVersion
}

// ValidHeaderVersion - valid incoming block version
func ValidHeaderVersion(currentVersion uint16, incomingVersion uint16) error {
	if incomingVersion < initialVersion {
		return fault.ErrInvalidBlockHeaderVersion
	}

	// incoming block version must be the same or higher than previous version
	if currentVersion > incomingVersion {
		return fault.ErrBlockVersionMustNotDecrease
	}

	return nil
}

// ValidBlockLinkage - valid incoming block linkage
func ValidBlockLinkage(currentDigest blockdigest.Digest, incomingDigest blockdigest.Digest) error {
	if currentDigest != incomingDigest {
		return fault.ErrPreviousBlockDigestDoesNotMatch
	}

	return nil
}
