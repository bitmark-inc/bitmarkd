// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package receiver_test

import (
	"testing"
	"time"

	"github.com/bitmark-inc/bitmarkd/announce/receiver"
	peerlib "github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
)

func TestString(t *testing.T) {
	str1 := "/ip4/1.2.3.4/tcp/1234"
	ma1, _ := ma.NewMultiaddr(str1)
	str2 := "/ip6/::1/tcp/5678"
	ma2, _ := ma.NewMultiaddr(str2)
	r := receiver.Receiver{
		ID:        peerlib.ID("this is a test"),
		Listeners: []ma.Multiaddr{ma1, ma2},
		Timestamp: time.Now(),
	}

	actual := r.String()

	assert.Equal(t, 2, len(actual), "wrong count")
	assert.Equal(t, str1, actual[0], "wrong first addr")
	assert.Equal(t, str2, actual[1], "wrong second addr")
}
