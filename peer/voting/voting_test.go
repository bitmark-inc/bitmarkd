// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package voting

import (
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/peer/mocks"
	"github.com/bitmark-inc/logger"
)

const (
	testHeight     = uint64(123)
	largerHeight   = uint64(456)
	testingDirName = "testing"
)

var (
	defaultDigest = blockdigest.Digest{
		0xf8, 0xb6, 0x16, 0x4d,
		0x19, 0xe2, 0xf6, 0x5a,
		0x2a, 0xae, 0x44, 0x8f,
		0x78, 0x7f, 0xe6, 0x6d,
		0x61, 0xe5, 0x7a, 0x48,
		0xc0, 0xc6, 0x77, 0x1b,
		0x1e, 0x92, 0x0b, 0x44,
		0x00, 0x00, 0x00, 0x00,
	}
	smallerDigest = blockdigest.Digest{
		0x11, 0xb6, 0x16, 0x4d,
		0x19, 0xe2, 0xf6, 0x5a,
		0x2a, 0xae, 0x44, 0x8f,
		0x78, 0x7f, 0xe6, 0x6d,
		0x61, 0xe5, 0x7a, 0x48,
		0xc0, 0xc6, 0x77, 0x1b,
		0x1e, 0x92, 0x0b, 0x44,
		0x00, 0x00, 0x00, 0x00,
	}
)

func setupTestLogger() {
	removeFiles()
	_ = os.Mkdir(testingDirName, 0o700)

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

func teardownTestLogger() {
	removeFiles()
}

func removeFiles() {
	os.RemoveAll(testingDirName)
}

func newTestVotingUpstream(t *testing.T) (*gomock.Controller, *mocks.MockUpstream) {
	ctl := gomock.NewController(t)
	return ctl, mocks.NewMockUpstream(ctl)
}

func newTestVotingZmqClient(t *testing.T) (*gomock.Controller, *mocks.MockClient) {
	ctl := gomock.NewController(t)
	return ctl, mocks.NewMockClient(ctl)
}

func newTestVoting() *VotingData {
	setupTestLogger()
	return &VotingData{
		votes:  make(records),
		result: &electionResult{},
		log:    logger.New("testVoting"),
	}
}

func TestSetMinHeight(t *testing.T) {
	v := newTestVoting()
	defer teardownTestLogger()

	v.SetMinHeight(testHeight)
	assert.Equal(t, testHeight, v.minHeight, "height not set")
}

func TestNumVoteOfDigestWhenOverMinHeight(t *testing.T) {
	v := newTestVoting()
	defer teardownTestLogger()

	ctl, mock := newTestVotingUpstream(t)
	defer ctl.Finish()

	e := &voters{
		candidate: mock,
		height:    testHeight,
	}

	numVote := v.NumVoteOfDigest(defaultDigest)
	assert.Equal(t, 0, numVote, "wrong election result")

	v.votes[defaultDigest] = []*voters{e}

	numVote = v.NumVoteOfDigest(defaultDigest)
	assert.Equal(t, 1, numVote, "wrong election result")
}

func TestVoteByWhenAboveMinHeight(t *testing.T) {
	v := NewVoting()
	ctl, mock := newTestVotingUpstream(t)
	defer ctl.Finish()

	//ctl2, mock2 := newTestVotingZmqClient(t)
	//defer ctl2.Finish()
	//mock2.EXPECT().String().Return("testing").Times(2)

	mock.EXPECT().RemoteAddr().Return("testing", nil).Times(2)
	mock.EXPECT().CachedRemoteHeight().Return(testHeight).Times(1)
	mock.EXPECT().CachedRemoteDigestOfLocalHeight().Return(defaultDigest).Times(1)
	//mock.EXPECT().Client().Return(mock2).Times(2)
	mock.EXPECT().CachedRemoteHeight().Return(largerHeight).Times(1)
	mock.EXPECT().CachedRemoteDigestOfLocalHeight().Return(defaultDigest).Times(1)
	mock.EXPECT().Name().Return("test").Times(2)

	v.SetMinHeight(testHeight)
	v.VoteBy(mock)

	numVote := v.NumVoteOfDigest(defaultDigest)
	assert.Equal(t, 1, numVote, "vote not count")

	v.VoteBy(mock)

	numVote = v.NumVoteOfDigest(defaultDigest)
	assert.Equal(t, 2, numVote, "vote not count")
}

func TestVoteByWhenBelowMinHeight(t *testing.T) {
	v := NewVoting()

	ctl, mock := newTestVotingUpstream(t)
	defer ctl.Finish()

	mock.EXPECT().RemoteAddr().Return("testing", nil).Times(1)
	mock.EXPECT().CachedRemoteHeight().Return(testHeight).Times(1)
	mock.EXPECT().CachedRemoteDigestOfLocalHeight().Return(defaultDigest).Times(1)
	mock.EXPECT().Name().Return("test").Times(1)

	v.SetMinHeight(largerHeight)
	v.VoteBy(mock)

	numVote := v.NumVoteOfDigest(defaultDigest)
	assert.Equal(t, 0, numVote, "vote count height below minimum")
}

func TestElectedCandidateWhenMajority(t *testing.T) {
	v := newTestVoting()
	defer teardownTestLogger()

	ctl3, mockZmq := newTestVotingZmqClient(t)
	mockZmq.EXPECT().String().Return("testing").AnyTimes()
	defer ctl3.Finish()

	ctl1, mock1 := newTestVotingUpstream(t)
	mock1.EXPECT().CachedRemoteHeight().Return(testHeight).Times(1)
	defer ctl1.Finish()

	ctl2, mock2 := newTestVotingUpstream(t)
	defer ctl2.Finish()

	// shorter chain with more votes
	e1 := &voters{
		candidate: mock1,
		height:    testHeight,
	}

	// longer chain with less votes
	e2 := &voters{
		candidate: mock2,
		height:    largerHeight,
	}

	v.votes[defaultDigest] = []*voters{e1, e1, e1}
	v.votes[smallerDigest] = []*voters{e2, e2}

	elected, height, err := v.ElectedCandidate()
	assert.Equal(t, nil, err, "should not exist error message")
	assert.Equal(t, testHeight, height, "majority not chosen")
	assert.Equal(t, mock1, elected, "wrong candidate")
}

func TestElectedCandidateWhenInSufficient(t *testing.T) {
	v := newTestVoting()
	defer teardownTestLogger()

	ctl3, _ := newTestVotingZmqClient(t)
	defer ctl3.Finish()

	ctl1, mock1 := newTestVotingUpstream(t)
	defer ctl1.Finish()

	ctl2, mock2 := newTestVotingUpstream(t)
	defer ctl2.Finish()

	// shorter chain with more votes
	e1 := &voters{
		candidate: mock1,
		height:    testHeight,
	}

	// longer chain with less votes
	e2 := &voters{
		candidate: mock2,
		height:    largerHeight,
	}

	v.votes[defaultDigest] = []*voters{e1, e1}
	v.votes[smallerDigest] = []*voters{e2}

	_, _, err := v.ElectedCandidate()
	assert.NotEqual(t, nil, err, "insufficient votes w/o error")
}

func TestElectedCandidateWhenDraw(t *testing.T) {
	v := newTestVoting()
	defer teardownTestLogger()

	clientStr := "test"
	ctl1, mock1 := newTestVotingUpstream(t)
	defer ctl1.Finish()
	mock1.EXPECT().CachedRemoteDigestOfLocalHeight().Return(smallerDigest).AnyTimes()
	mock1.EXPECT().RemoteAddr().Return(clientStr, nil).AnyTimes()
	mock1.EXPECT().CachedRemoteHeight().Return(testHeight).Times(1)

	ctl2, mock2 := newTestVotingUpstream(t)
	defer ctl2.Finish()
	mock2.EXPECT().CachedRemoteDigestOfLocalHeight().Return(defaultDigest).AnyTimes()
	mock2.EXPECT().RemoteAddr().Return(clientStr, nil).AnyTimes()

	e1 := &voters{
		candidate: mock1,
		height:    testHeight,
	}
	e2 := &voters{
		candidate: mock2,
		height:    largerHeight,
	}

	v.votes[defaultDigest] = []*voters{e2, e2}
	v.votes[smallerDigest] = []*voters{e1, e1}

	elected, height, err := v.ElectedCandidate()
	assert.Equal(t, nil, err, "wrong error message")
	assert.Equal(t, testHeight, height, "wrong height")
	assert.Equal(t, smallerDigest, elected.CachedRemoteDigestOfLocalHeight(), "wrong digest candidate")
}

func TestReset(t *testing.T) {
	v := newTestVoting()
	defer teardownTestLogger()

	ctl, mock := newTestVotingUpstream(t)
	defer ctl.Finish()
	e := &voters{
		candidate: mock,
		height:    testHeight,
	}

	v.SetMinHeight(testHeight)
	v.votes[defaultDigest] = []*voters{e}
	v.Reset()

	numVote := v.NumVoteOfDigest(defaultDigest)
	assert.Equal(t, 0, numVote, "votes not reset")
	assert.Equal(t, uint64(0), v.minHeight, "minHeight not reset")
	assert.Equal(t, electionResult{
		highestNumVotes: 0,
		winner:          nil,
		majorityHeight:  uint64(0),
		draw:            false,
	}, *v.result, "election result not reset")
}
