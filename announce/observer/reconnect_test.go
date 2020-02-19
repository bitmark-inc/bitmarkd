// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package observer_test

import (
	"testing"

	"github.com/bitmark-inc/bitmarkd/announce/observer"

	"github.com/bitmark-inc/bitmarkd/announce/mocks"

	"github.com/golang/mock/gomock"
)

func TestReconnectUpdate(t *testing.T) {
	ctl := gomock.NewController(t)
	m := mocks.NewMockReceptor(ctl)
	defer ctl.Finish()

	m.EXPECT().ReBalance().Return().Times(1)

	r := observer.NewReconnect(m)
	r.Update("self", [][]byte{})
}

func TestReconnectUpdateWhenEventNotMatch(t *testing.T) {
	ctl := gomock.NewController(t)
	m := mocks.NewMockReceptor(ctl)
	defer ctl.Finish()

	m.EXPECT().ReBalance().Return().Times(0)

	r := observer.NewReconnect(m)
	r.Update("not_reconnect", [][]byte{})
}
