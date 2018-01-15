// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpccalls

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/keypair"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/bitmarkd/rpc"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/sha3"
	"io"
	"time"
)

var (
	ErrMakeIssueFail = fault.ProcessError("make issue failed")
)

type IssueData struct {
	Issuer     *keypair.KeyPair
	AssetIndex *transactionrecord.AssetIndex
	Quantity   int
}

// JSON data to output after asset/issue/proof completes
type IssueReply struct {
	AssetId        transactionrecord.AssetIndex `json:"assetId"`
	IssueIds       []merkle.Digest              `json:"issueIds"`
	PayId          pay.PayId                    `json:"payId"`
	PayNonce       reservoir.PayNonce           `json:"payNonce"`
	Difficulty     string                       `json:"difficulty"`
	SubmittedNonce string                       `json:"submittedNonce"`
	ProofStatus    reservoir.TrackingStatus     `json:"proofStatus"`
}

func (client *Client) Issue(issueConfig *IssueData) (*IssueReply, error) {

	nonce := time.Now().UTC().Unix() * 1000
	issues := make([]*transactionrecord.BitmarkIssue, issueConfig.Quantity)
	for i := 0; i < len(issues); i += 1 {
		issue, err := makeIssue(client.testnet, issueConfig, uint64(nonce)+uint64(i))
		if nil != err {
			return nil, err
		}
		if nil == issue {
			return nil, ErrMakeIssueFail
		}
		issues[i] = issue
	}

	client.printJson("Issue Request", issues)

	issuesArgs := rpc.CreateArguments{
		Assets: nil,
		Issues: issues,
	}

	var issuesReply rpc.CreateReply
	if err := client.client.Call("Bitmarks.Create", issuesArgs, &issuesReply); err != nil {
		return nil, err
	}

	client.printJson("Issue Reply", issuesReply)

	// run proofer to generate local nonce
	localNonce := makeProof(issuesReply.PayId, issuesReply.PayNonce, issuesReply.Difficulty, client.verbose, client.handle)
	proofArgs := rpc.ProofArguments{
		PayId: issuesReply.PayId,
		Nonce: localNonce,
	}

	client.printJson("Proof Request", proofArgs)

	var proofReply rpc.ProofReply
	if err := client.client.Call("Bitmarks.Proof", &proofArgs, &proofReply); err != nil {
		return nil, err
	}

	client.printJson("Proof Reply", proofReply)

	// make response
	response := IssueReply{
		AssetId:        issues[0].AssetIndex, // Note: all issues are for the same asset
		IssueIds:       make([]merkle.Digest, len(issues)),
		PayId:          issuesReply.PayId,
		PayNonce:       issuesReply.PayNonce,
		Difficulty:     issuesReply.Difficulty,
		SubmittedNonce: proofArgs.Nonce,
		ProofStatus:    proofReply.Status,
	}

	for i := 0; i < len(issuesReply.Issues); i++ {
		response.IssueIds[i] = issuesReply.Issues[i].TxId
	}

	return &response, nil
}

// build a properly signed issues
func makeIssue(testnet bool, issueConfig *IssueData, nonce uint64) (*transactionrecord.BitmarkIssue, error) {

	issuerAddress := makeAddress(issueConfig.Issuer, testnet)

	r := transactionrecord.BitmarkIssue{
		AssetIndex: *issueConfig.AssetIndex,
		Owner:      issuerAddress,
		Nonce:      nonce,
		Signature:  nil,
	}

	// pack without signature
	packed, err := r.Pack(issuerAddress)
	if nil == err {
		return nil, ErrMakeTransferFail
	} else if fault.ErrInvalidSignature != err {
		return nil, err
	}

	// manually sign the record and attach signature
	signature := ed25519.Sign(issueConfig.Issuer.PrivateKey, packed)
	r.Signature = signature[:]

	// check that signature is correct by packing again
	_, err = r.Pack(issuerAddress)
	if nil != err {
		return nil, err
	}
	return &r, nil
}

// determine the nonce as a hex string
func makeProof(payId pay.PayId, payNonce reservoir.PayNonce, difficulty string, verbose bool, handle io.Writer) string {

	nonce := uint64(12345)
	nonceBuffer := make([]byte, 8)

	start := time.Now()
	hashCount := 0

	for {
		hashCount += 1
		nonce += 113113
		binary.BigEndian.PutUint64(nonceBuffer, nonce)

		// compute hash
		h := sha3.New256()
		h.Write(payId[:])
		h.Write(payNonce[:])
		h.Write(nonceBuffer)
		var digest [32]byte
		h.Sum(digest[:0])
		if 0 == digest[0]|digest[1] {
			if verbose {
				hps := float64(hashCount) / time.Now().Sub(start).Seconds() / 1.0e6
				fmt.Fprintf(handle, "%f MH/s: possible nonce: %x  with digest: %x\n", hps, nonceBuffer, digest)
			}
			hexDigest := hex.EncodeToString(digest[:])
			if hexDigest <= difficulty {
				return hex.EncodeToString(nonceBuffer)
			}
		}
	}
}
