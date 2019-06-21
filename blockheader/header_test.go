// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockheader_test

import (
	"testing"
	"time"

	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/blockheader"
)

func TestHeader(t *testing.T) {
	setup(t)
	defer teardown(t)

	someHeight := uint64(1234567890)
	someDigest := blockdigest.Digest{
		0x2b, 0xa1, 0x2b, 0xa1, 0x54, 0x2b, 0xa1, 0x54,
		0x14, 0x46, 0x74, 0x29, 0x1d, 0x29, 0x1d, 0x29,
		0x2b, 0xa1, 0x2b, 0xa1, 0x54, 0x2b, 0xa1, 0x54,
		0x14, 0x46, 0x74, 0x29, 0x1d, 0x29, 0x1d, 0x29,
	}
	someVersion := uint16(15)
	someTimestamp := uint64(time.Now().Unix())

	blockheader.Set(someHeight, someDigest, someVersion, someTimestamp)

	height, digest, version, timestamp := blockheader.Get()

	if height != someHeight {
		t.Errorf("height: actual: %d  expected: %d", height, someHeight)
	}
	if digest != someDigest {
		t.Errorf("digest: actual: %d  expected: %d", digest, someDigest)
	}
	if version != someVersion {
		t.Errorf("version: actual: %d  expected: %d", version, someVersion)
	}
	if timestamp != someTimestamp {
		t.Errorf("timestamp: actual: %d  expected: %d", timestamp, someTimestamp)
	}

	digest, height = blockheader.GetNew()

	if digest != someDigest {
		t.Errorf("digest: actual: %d  expected: %d", digest, someDigest)
	}

	if height != someHeight+1 {
		t.Errorf("height: actual: %d  expected: %d", height, someHeight+1)
	}

	height = blockheader.Height()
	if height != someHeight {
		t.Errorf("height: actual: %d  expected: %d", height, someHeight)
	}
}
