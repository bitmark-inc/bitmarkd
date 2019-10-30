// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package voting

import (
	"fmt"

	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/logger"
)

const (
	loggerCategory = "voting"
	minimumClients = 5
)

//Candidate interface for candidate
type Candidate interface {
	CachedRemoteHeight() uint64
	CachedRemoteDigestOfLocalHeight() blockdigest.Digest
	RemoteAddr() string
	Name() string
}

//Voting interface for voting
type Voting interface {
	ElectedCandidate() (Candidate, uint64, error)
	NumVoteOfDigest(blockdigest.Digest) int
	Reset()
	SetMinHeight(uint64)
	VoteBy(Candidate)
}

// each voter is also a candidate, it means all candidates vote
// to height itself has
type voters struct {
	candidate Candidate
	height    uint64
}

type records map[blockdigest.Digest][]*voters

type electionResult struct {
	highestNumVotes int
	majorityHeight  uint64
	winner          Candidate
	draw            bool
}

//VotingImpl Implementation of Voting
type VotingImpl struct {
	votes     records
	minHeight uint64
	result    *electionResult
	log       *logger.L
}

// NewVoting - new voting object
func NewVoting() Voting {
	return &VotingImpl{
		votes:     make(records),
		minHeight: uint64(0),
		result: &electionResult{
			highestNumVotes: 0,
			winner:          nil,
			majorityHeight:  uint64(0),
			draw:            false,
		},
		log: logger.New(loggerCategory),
	}
}

// SetMinHeight - set minimum height for vote
func (v *VotingImpl) SetMinHeight(height uint64) {
	v.minHeight = height
}

// NumVoteOfDigest - number of votest for a digest
func (v *VotingImpl) NumVoteOfDigest(digest blockdigest.Digest) int {
	if v.existVoteForDigest(digest) {
		return len(v.votes[digest])
	}
	return 0
}

func (v *VotingImpl) existVoteForDigest(digest blockdigest.Digest) bool {
	_, ok := v.votes[digest]
	return ok
}

// VoteBy - vote by some upstream
func (v *VotingImpl) VoteBy(candidate Candidate) {
	height := candidate.CachedRemoteHeight()
	digest := candidate.CachedRemoteDigestOfLocalHeight()
	remoteAddr := candidate.RemoteAddr()
	remoteName := candidate.Name()

	v.log.Debugf(
		"\x1b[32m%s connects to remote %s, cached remote height: %d with digest: %s\x1b[0m",
		remoteName,
		remoteAddr,
		height,
		digest.String(),
	)

	if !v.validHeight(height) {
		v.log.Infof(
			"\x1b[32mremote cached height: %d, below minimum height %d, discard\x1b[0m",
			height,
			v.minHeight,
		)
		return
	}

	e := &voters{
		candidate: candidate,
		height:    height,
	}

	if v.existVoteForDigest(digest) {
		v.votes[digest] = append(v.votes[digest], e)
		return
	}
	v.log.Debugf("\x1b[32m%s connect to remote %s, vote success\x1b[0m", candidate.Name(), remoteAddr)
	v.votes[digest] = []*voters{e}
}

func (v *VotingImpl) validHeight(height uint64) bool {
	return height >= v.minHeight
}

// ElectedCandidate - get candidate that is most vote
func (v *VotingImpl) ElectedCandidate() (Candidate, uint64, error) {
	err := v.countVotes()
	if nil != err {
		v.log.Errorf("count votes with error: %s", err)
		return nil, uint64(0), err
	}

	if v.result.draw {
		return v.drawElection()
	}
	return v.result.winner, v.result.winner.CachedRemoteHeight(), nil
}

func (v *VotingImpl) countVotes() error {
	for _, voters := range v.votes {
		if v.result.highestNumVotes < len(voters) {
			v.updateTemporarilyVoteSummary(voters)
		} else if v.result.highestNumVotes == len(voters) {
			v.result.draw = true
		}
	}
	v.log.Debugf(
		"vote draw: %t, most votes: %d, majority height: %d",
		v.result.draw,
		v.result.highestNumVotes,
		v.result.majorityHeight,
	)

	if nil == v.result.winner {
		return fmt.Errorf("%s", fault.ErrVotesEmptyWinner)
	}

	if !v.sufficientVotes() {
		return fmt.Errorf("%s", fault.ErrVotesInsufficient)
	}

	return nil
}

func (v *VotingImpl) sufficientVotes() bool {
	if !v.result.draw {
		return v.result.highestNumVotes >= (1+minimumClients)/2
	}

	return v.sufficientVotesInDraw()
}

func (v *VotingImpl) sufficientVotesInDraw() bool {
	drawVotes := 0
	for _, voters := range v.votes {
		counts := len(voters)
		if v.result.highestNumVotes == counts {
			drawVotes += counts
		}
	}
	return drawVotes >= (1+minimumClients)/2
}

func (v *VotingImpl) updateTemporarilyVoteSummary(voters []*voters) {
	v.result.highestNumVotes = len(voters)
	v.result.winner = voters[0].candidate
	v.result.majorityHeight = voters[0].height
	v.result.draw = false
}

// when in draw, which chain has smaller digest is chosen
// compare by remote digest of local height
func (v *VotingImpl) drawElection() (Candidate, uint64, error) {
	var err error

	v.log.Infof("election in draw with vote counts %d", v.result.highestNumVotes)
	v.result.winner, err = v.drawWinner()
	if nil != err {
		return nil, uint64(0), err
	}
	return v.result.winner, v.result.winner.CachedRemoteHeight(), nil
}

func (v *VotingImpl) drawWinner() (Candidate, error) {
	if 0 == v.result.highestNumVotes {
		return nil, fault.ErrVotesZeroCount
	}

	if uint64(0) == v.result.majorityHeight {
		return nil, fault.ErrVotesZeroHeight
	}
	v.log.Debug("start to decide which is best winner")
	candidates := v.sameVoteCandidates(v.result.highestNumVotes)
	elected := v.smallerDigestWinnerFrom(candidates)

	return elected, nil
}

func (v *VotingImpl) sameVoteCandidates(numVote int) []Candidate {
	var candidates []Candidate
	for _, elections := range v.votes {
		if len(elections) == numVote {
			candidates = append(candidates, elections[0].candidate)
		}
	}

	v.log.Infof("same vote with possible candates: %+v", candidates)
	return candidates
}

func (v *VotingImpl) smallerDigestWinnerFrom(
	candidates []Candidate,
) Candidate {
	v.log.Debug("select candidate with smaller digest")

	elected := candidates[0]
election:
	for i := 1; i < len(candidates); i++ {
		targetDigest := candidates[i].CachedRemoteDigestOfLocalHeight()
		electedDigest := elected.CachedRemoteDigestOfLocalHeight()
		if v.allZeros(electedDigest) || v.allZeros(targetDigest) {
			continue election
		}
		if !electedDigest.SmallerDigestThan(targetDigest) {
			v.log.Debugf("digest %v is larger than %v", electedDigest, targetDigest)
			elected = candidates[i]
		} else {
			v.log.Debugf("digest %v is smaller than %v", electedDigest, targetDigest)
		}
	}
	v.log.Infof(
		"digest %x is the smallest among all",
		elected.CachedRemoteDigestOfLocalHeight(),
	)
	return elected
}

func (v *VotingImpl) allZeros(s blockdigest.Digest) bool {
	for _, b := range s {
		if b != 0 {
			return false
		}
	}
	return true
}

// Reset - reset voting
func (v *VotingImpl) Reset() {
	v.votes = make(records)
	v.minHeight = uint64(0)
	v.resetResult()
}

func (v *VotingImpl) resetResult() {
	v.result.highestNumVotes = 0
	v.result.winner = nil
	v.result.majorityHeight = uint64(0)
	v.result.draw = false
}
