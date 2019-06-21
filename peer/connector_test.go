// SPDX-License-Identifier: ISC
package peer

import (
	"testing"

	"github.com/bitmark-inc/bitmarkd/peer/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func newTestConnector() *connector {
	return &connector{}
}

func newTestMockUpstream(t *testing.T) (*gomock.Controller, *mocks.MockUpstreamIntf) {
	ctl := gomock.NewController(t)
	return ctl, mocks.NewMockUpstreamIntf(ctl)
}

func TestNextState(t *testing.T) {
	c := newTestConnector()
	orig := c.state
	c.nextState()
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
