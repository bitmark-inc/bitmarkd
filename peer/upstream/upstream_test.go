// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package upstream

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/peer/mocks"
	"github.com/bitmark-inc/logger"
)

const (
	testingDirName      = "testing"
	testLoggerName      = "testUpstream"
	defaultStringDigest = "12345678901234567890123456789012"
)

var (
	defaultDigest blockdigest.Digest
)

func init() {
	fmt.Sscan(defaultStringDigest, &defaultDigest)
}

func setupTestUpstreamLogger() {
	removeFiles()
	_ = os.Mkdir(testingDirName, 0700)

	logging := logger.Configuration{
		Directory: testingDirName,
		File:      "testing.log",
		Size:      1048576,
		Count:     10,
		Console:   false,
		Levels: map[string]string{
			logger.DefaultTag: "critical",
		},
	}

	// start logging
	_ = logger.Initialise(logging)
}

func removeFiles() {
	os.RemoveAll(testingDirName)
}

func teardownTestUpstreamLogger() {
	removeFiles()
}

func newMockZmqutilClient(t *testing.T) (*gomock.Controller, *mocks.MockClient) {
	ctl := gomock.NewController(t)
	return ctl, mocks.NewMockClient(ctl)
}

func newTestUpstream(t *testing.T) (Upstream, *gomock.Controller, *mocks.MockClient) {
	setupTestUpstreamLogger()
	ctl, mockZmq := newMockZmqutilClient(t)

	return &upstreamData{
		log:                       logger.New(testLoggerName),
		client:                    mockZmq,
		remoteDigestOfLocalHeight: defaultDigest,
	}, ctl, mockZmq
}

func TestCachedRemoteDigestOfLocalHeight(t *testing.T) {
	u, ctl, _ := newTestUpstream(t)
	_ = announce.Initialise("test", "")
	defer ctl.Finish()
	defer teardownTestUpstreamLogger()
	defer announce.Finalise()

	actual := u.CachedRemoteDigestOfLocalHeight()
	assert.Equal(t, defaultDigest, actual, "wrong digest")
}

func TestGetClientStringWhenNoClient(t *testing.T) {
	setupTestUpstreamLogger()
	u := &upstreamData{
		log: logger.New(testLoggerName),
	}
	defer teardownTestUpstreamLogger()

	str, err := u.RemoteAddr()
	assert.Equal(t, "", str, "wrong client string")
	assert.NotEqual(t, nil, err, "error is empty")
}

func TestGetClientStringWhenValid(t *testing.T) {
	u, ctl, mock := newTestUpstream(t)
	defer ctl.Finish()

	clientStr := "test"
	mock.EXPECT().IsConnected().Return(true).Times(1)
	mock.EXPECT().String().Return(clientStr).Times(1)

	str, err := u.RemoteAddr()
	assert.Equal(t, clientStr, str, "wrong client string")
	assert.Equal(t, nil, err, "error not empty")
}

func TestGetClientStringWhenNotConnected(t *testing.T) {
	u, ctl, mock := newTestUpstream(t)
	defer ctl.Finish()

	mock.EXPECT().IsConnected().Return(false).Times(1)

	str, err := u.RemoteAddr()
	assert.Equal(t, "", str, "wrong client string")
	assert.NotEqual(t, nil, err, "error is empty")
}

func TestActiveInPastSecondsWhenInRange(t *testing.T) {
	setupTestUpstreamLogger()
	u := &upstreamData{
		log: logger.New(testLoggerName),
	}
	defer teardownTestUpstreamLogger()

	now := time.Now()
	u.lastResponseTime = now.Add(-5 * time.Second)

	actual := u.ActiveInThePast(30 * time.Second)
	assert.Equal(t, true, actual, "wrong time range")
}

func TestActiveInPastSecondsWhenOutOfRange(t *testing.T) {
	setupTestUpstreamLogger()
	u := &upstreamData{
		log: logger.New(testLoggerName),
	}
	defer teardownTestUpstreamLogger()

	now := time.Now()
	fiveSecBefore := now.Add(-35 * time.Second)
	u.lastResponseTime = fiveSecBefore
	actual := u.ActiveInThePast(30 * time.Second)
	assert.Equal(t, false, actual, "wrong time range")
}

func TestCachedRemoteHeight(t *testing.T) {
	setupTestUpstreamLogger()
	u := &upstreamData{
		log: logger.New(testLoggerName),
	}
	defer teardownTestUpstreamLogger()

	height := uint64(100)
	u.remoteHeight = uint64(height)
	actual := u.CachedRemoteHeight()
	assert.Equal(t, height, actual, "wrong cached height")
}

func TestName(t *testing.T) {
	setupTestUpstreamLogger()
	u := &upstreamData{
		log: logger.New(testLoggerName),
	}
	defer teardownTestUpstreamLogger()
	name := "testing"

	u.name = name
	actual := u.Name()
	assert.Equal(t, name, actual, "wrong name")
}

func TestLocalHeight(t *testing.T) {
	setupTestUpstreamLogger()
	u := &upstreamData{
		log: logger.New(testLoggerName),
	}
	defer teardownTestUpstreamLogger()

	height := uint64(100)
	u.localHeight = height
	actual := u.LocalHeight()
	assert.Equal(t, height, actual, "wrong local height")
}
