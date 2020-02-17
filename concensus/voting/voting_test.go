// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package voting

import (
	"os"
	"testing"

	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/concensus/mocks"
	"github.com/bitmark-inc/logger"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
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

func teardownTestLogger() {
	removeFiles()
}

func removeFiles() {
	_ = os.RemoveAll(testingDirName)
}

func newTestVoting() Voting {
	return NewVoting()
}

func newTestVotingCandidate(t *testing.T) (*gomock.Controller, *mocks.MockCandidate) {
	ctl := gomock.NewController(t)
	return ctl, mocks.NewMockCandidate(ctl)
}

func newTestVotingImpl() *VotingImpl {
	setupTestLogger()
	return &VotingImpl{
		votes:  make(records),
		result: &electionResult{},
		log:    logger.New("testVoting"),
	}
}

func TestSetMinHeight(t *testing.T) {
	v := newTestVotingImpl()
	defer teardownTestLogger()

	v.SetMinHeight(testHeight)
	assert.Equal(t, testHeight, v.minHeight, "height not set")
}

func TestNumVoteOfDigestWhenOverMinHeight(t *testing.T) {
	v := newTestVotingImpl()
	defer teardownTestLogger()

	ctl, mock := newTestVotingCandidate(t)
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

func TestVoteByWhenOverMinHeight(t *testing.T) {
	v := newTestVoting()
	ctl, mock := newTestVotingCandidate(t)
	defer ctl.Finish()

	mock.EXPECT().CachedRemoteHeight().Return(testHeight).Times(1)
	mock.EXPECT().CachedRemoteDigestOfLocalHeight().Return(defaultDigest).Times(1)
	mock.EXPECT().RemoteAddr().Return("addr").Times(2)
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
	v := newTestVoting()

	ctl, mock := newTestVotingCandidate(t)
	defer ctl.Finish()

	mock.EXPECT().CachedRemoteHeight().Return(testHeight).Times(1)
	mock.EXPECT().CachedRemoteDigestOfLocalHeight().Return(defaultDigest).Times(1)
	mock.EXPECT().RemoteAddr().Return("addr").Times(1)
	mock.EXPECT().Name().Return("test").Times(1)

	v.SetMinHeight(largerHeight)
	v.VoteBy(mock)

	numVote := v.NumVoteOfDigest(defaultDigest)
	assert.Equal(t, 0, numVote, "vote count height below minimum")
}

func TestElectedCandidateWhenMajority(t *testing.T) {
	v := newTestVotingImpl()
	defer teardownTestLogger()

	//ctl3, mockZmq := newTestVotingZmqClient(t)
	//mockZmq.EXPECT().String().Return("testing").AnyTimes()
	//defer ctl3.Finish()

	ctl1, mock1 := newTestVotingCandidate(t)
	//mock1.EXPECT().Client().Return(mockZmq).AnyTimes()
	mock1.EXPECT().CachedRemoteHeight().Return(testHeight).Times(1)
	defer ctl1.Finish()

	ctl2, mock2 := newTestVotingCandidate(t)
	//mock2.EXPECT().Client().Return(mockZmq).AnyTimes()
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
	v := newTestVotingImpl()
	defer teardownTestLogger()

	//ctl3, _ := newTestVotingZmqClient(t)
	//defer ctl3.Finish()

	ctl1, mock1 := newTestVotingCandidate(t)
	defer ctl1.Finish()

	ctl2, mock2 := newTestVotingCandidate(t)
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
	v := newTestVotingImpl()
	defer teardownTestLogger()

	ctl1, mock1 := newTestVotingCandidate(t)
	defer ctl1.Finish()
	mock1.EXPECT().CachedRemoteDigestOfLocalHeight().Return(smallerDigest).AnyTimes()
	mock1.EXPECT().CachedRemoteHeight().Return(testHeight).Times(1)

	ctl2, mock2 := newTestVotingCandidate(t)
	defer ctl2.Finish()
	mock2.EXPECT().CachedRemoteDigestOfLocalHeight().Return(defaultDigest).AnyTimes()

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
	v := newTestVotingImpl()
	defer teardownTestLogger()

	ctl, mock := newTestVotingCandidate(t)
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
