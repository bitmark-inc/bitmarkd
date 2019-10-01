// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package peer

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/bitmarkd/peer/mocks"
)

func newTestConnector() *connector {
	return &connector{}
}

func newTestMockUpstream(t *testing.T) (*gomock.Controller, *mocks.MockUpstream) {
	ctl := gomock.NewController(t)
	return ctl, mocks.NewMockUpstream(ctl)
}

func TestNextState(t *testing.T) {
	c := newTestConnector()
	orig := c.state
	c.nextState(orig + 1)
	assert.Equal(t, orig+1, c.state, "state not increased")
}

func TestGetConnectedClientCount(t *testing.T) {
	c := newTestConnector()
	ctl, mockUpstream := newTestMockUpstream(t)
	defer ctl.Finish()

	mockUpstream.EXPECT().IsConnected().Return(true).Times(1)
	mockUpstream.EXPECT().IsConnected().Return(false).Times(1)

	c.dynamicClients.PushBack(mockUpstream)
	actual := c.getConnectedClientCount()
	assert.Equal(t, 1, actual, "wrong connected client count")

	actual = c.getConnectedClientCount()
	assert.Equal(t, 0, actual, "wrong connected client count")
}
