// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package discovery

import (
	"testing"
)

func TestRestorePeers(t *testing.T) {
	// if peer file not exist, do not show any error
	notExistFile := "file_not_exist.json"
	err := restorePeers(notExistFile)
	if nil != err {
		t.Errorf("Peer file not exist should not return error.")
	}
}
