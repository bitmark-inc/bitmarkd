// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package voting

import (
	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/peer/upstream"
	"github.com/bitmark-inc/logger"
)

const (
	loggerCategory = "voting"
	minimumClients = 5
)

type Voting interface {
	ElectedCandidate() (upstream.Upstream, uint64, error)
	NumVoteOfDigest(blockdigest.Digest) int
	Reset()
	SetMinHeight(uint64)
	VoteBy(upstream.Upstream)
}

// each voter is also a candidate, it means all candidates vote
// to height itself has
type voters struct {
	candidate upstream.Upstream
	height    uint64
}

type records map[blockdigest.Digest][]*voters

type electionResult struct {
	highestNumVotes int
	majorityHeight  uint64
	winner          upstream.Upstream
	draw            bool
}

type VotingData struct {
	Voting

	votes     records
	minHeight uint64
	result    *electionResult
	log       *logger.L
}

// NewVoting - new voting object
func NewVoting() Voting {
	return &VotingData{
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
func (v *VotingData) SetMinHeight(height uint64) {
	v.log.Infof("minimum height %d\n", height)
	v.minHeight = height
}

// NumVoteOfDigest - number of votest for a digest
func (v *VotingData) NumVoteOfDigest(digest blockdigest.Digest) int {
	if v.existVoteForDigest(digest) {
		return len(v.votes[digest])
	}
	return 0
}

func (v *VotingData) existVoteForDigest(digest blockdigest.Digest) bool {
	_, ok := v.votes[digest]
	return ok
}

// VoteBy - vote by some upstream
func (v *VotingData) VoteBy(candidate upstream.Upstream) {
	height := candidate.CachedRemoteHeight()
	digest := candidate.CachedRemoteDigestOfLocalHeight()
	upstream := candidate.Name()

	remoteAddr, err := candidate.RemoteAddr()
	if nil != err {
		v.log.Infof("remote addr error: %s", err)
	}

	v.log.Infof(
		"%s connects to remote %s, cached remote height: %d with digest: %s",
		upstream,
		remoteAddr,
		height,
		digest.String(),
	)

	if !v.validHeight(height) {
		v.log.Infof(
			"remote cached height: %d, below minimum height %d, discard",
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

	v.log.Debugf("%s connect to remote %s, vote success", upstream, remoteAddr)
	v.votes[digest] = []*voters{e}
}

func (v *VotingData) validHeight(height uint64) bool {
	return height >= v.minHeight
}

// ElectedCandidate - get candidate that is most vote
func (v *VotingData) ElectedCandidate() (upstream.Upstream, uint64, error) {
	err := v.countVotes()
	if nil != err {
		v.log.Warnf("count votes with error: %s", err)
		return nil, uint64(0), err
	}

	if v.result.draw {
		return v.drawElection()
	}

	return v.result.winner, v.result.winner.CachedRemoteHeight(), nil
}

func (v *VotingData) countVotes() error {
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
		return fault.VotesWithEmptyWinner
	}

	if !v.sufficientVotes() {
		return fault.VotesInsufficient
	}

	return nil
}

func (v *VotingData) sufficientVotes() bool {
	if !v.result.draw {
		return v.result.highestNumVotes >= (1+minimumClients)/2
	}

	return v.sufficientVotesInDraw()
}

func (v *VotingData) sufficientVotesInDraw() bool {
	drawVotes := 0
	for _, voters := range v.votes {
		counts := len(voters)
		if v.result.highestNumVotes == counts {
			drawVotes += counts
		}
	}
	return drawVotes >= (1+minimumClients)/2
}

func (v *VotingData) updateTemporarilyVoteSummary(voters []*voters) {
	v.result.highestNumVotes = len(voters)
	v.result.winner = voters[0].candidate
	v.result.majorityHeight = voters[0].height
	v.result.draw = false
}

// when in draw, which chain has smaller digest is chosen
// compare by remote digest of local height
func (v *VotingData) drawElection() (upstream.Upstream, uint64, error) {
	var err error

	v.log.Infof("election in draw with vote counts %d", v.result.highestNumVotes)
	v.result.winner, err = v.drawWinner()
	if nil != err {
		return nil, uint64(0), err
	}
	return v.result.winner, v.result.winner.CachedRemoteHeight(), nil
}

func (v *VotingData) drawWinner() (upstream.Upstream, error) {
	if 0 == v.result.highestNumVotes {
		return nil, fault.VotesWithZeroCount
	}

	if uint64(0) == v.result.majorityHeight {
		return nil, fault.VotesWithZeroHeight
	}
	v.log.Debug("start to decide which is best winner")
	candidates := v.sameVoteCandidates(v.result.highestNumVotes)
	elected := v.smallerDigestWinnerFrom(candidates)

	return elected, nil
}

func (v *VotingData) sameVoteCandidates(numVote int) []upstream.Upstream {
	var candidates []upstream.Upstream
	for _, elections := range v.votes {
		if len(elections) == numVote {
			candidates = append(candidates, elections[0].candidate)
		}
	}

	v.log.Infof("same vote with possible candates: %+v", candidates)
	return candidates
}

func (v *VotingData) smallerDigestWinnerFrom(
	candidates []upstream.Upstream,
) upstream.Upstream {
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

func (v *VotingData) allZeros(s blockdigest.Digest) bool {
	for _, b := range s {
		if b != 0 {
			return false
		}
	}
	return true
}

// Reset - reset voting
func (v *VotingData) Reset() {
	v.votes = make(records)
	v.minHeight = 0
	v.resetResult()
}

func (v *VotingData) resetResult() {
	v.result.highestNumVotes = 0
	v.result.winner = nil
	v.result.majorityHeight = 0
	v.result.draw = false
}
