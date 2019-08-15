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
	blockTimeSpacingInitialInSecond = 240 * 60
	blockTimeSpacingCurrentInSecond = 10 * 60

	initialVersion             = 1
	modifiedTimeSpacingVersion = 2
	difficultyAppliedVersion   = 4
)

// ValidBlockTimeSpacingAtVersion - valid block time spacing based on different version
func ValidBlockTimeSpacingAtVersion(version uint16, timeSpacing uint64) error {
	if version == initialVersion && timeSpacing > blockTimeSpacingInitialInSecond {
		return fault.ErrInvalidBlockHeaderTimestamp
	}

	if version >= modifiedTimeSpacingVersion && timeSpacing > blockTimeSpacingCurrentInSecond {
		return fault.ErrInvalidBlockHeaderTimestamp
	}

	return nil
}

// ValidIncomingDifficuty - valid incoming difficulty
func ValidIncomingDifficuty(header *Header) error {
	if !IsDifficultyAppliedVersion(header.Version) {
		return nil
	}

	if header.Difficulty.Value() != difficulty.Current.Value() {
		return fault.ErrDifficultyNotMatch
	}
	return nil
}

// IsDifficultyAppliedVersion - is difficulty rule applied at header version
func IsDifficultyAppliedVersion(version uint16) bool {
	return version >= difficultyAppliedVersion
}

// IsBlockToAdjustDifficulty - is block the one to adjust difficulty
func IsBlockToAdjustDifficulty(height uint64, version uint16) bool {
	if !IsDifficultyAppliedVersion(version) {
		return false
	}
	return isDifficultyAdjustmentBlock(height)
}

func isDifficultyAdjustmentBlock(height uint64) bool {
	return 0 == height%difficulty.AdjustTimespanInBlocks
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
func ValidBlockLinkage(currentDigest blockdigest.Digest, incomingDigestOfPreviousBlock blockdigest.Digest) error {
	if currentDigest != incomingDigestOfPreviousBlock {
		return fault.ErrPreviousBlockDigestDoesNotMatch
	}

	return nil
}

// most time next height is increased by 1 from current height,
// but it is also valid when incoming block is same height with smaller digest
func validNextHeightFromExpected(expectedNextHeight uint64, nextHeight uint64) error {
	if nextHeight < expectedNextHeight-1 || nextHeight > expectedNextHeight {
		return fault.ErrHeightOutOfSequence
	}

	return nil
}
