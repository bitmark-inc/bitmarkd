// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpccalls

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/sha3"

	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/configuration"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/bitmarkd/rpc"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
)

// IssueData - data for an issue request
type IssueData struct {
	Issuer    *configuration.Private
	AssetId   *transactionrecord.AssetIdentifier
	Quantity  int
	FreeIssue bool
}

// IssueReply - JSON data to output after asset/issue/proof completes
type IssueReply struct {
	AssetId        transactionrecord.AssetIdentifier               `json:"assetId"`
	IssueIds       []merkle.Digest                                 `json:"issueIds"`
	PayId          pay.PayId                                       `json:"payId"`
	PayNonce       reservoir.PayNonce                              `json:"payNonce"`
	Difficulty     string                                          `json:"difficulty"`
	SubmittedNonce string                                          `json:"submittedNonce"`
	ProofStatus    reservoir.TrackingStatus                        `json:"proofStatus"`
	Payments       map[string]transactionrecord.PaymentAlternative `json:"payments,omitempty"`
	Commands       map[string]string                               `json:"commands,omitempty"`
}

// Issue - perform an issue request
func (client *Client) Issue(issueConfig *IssueData) (*IssueReply, error) {

	if issueConfig.FreeIssue && 1 != issueConfig.Quantity {
		return nil, fmt.Errorf("quantity: %d > 1 is not allowed for free", issueConfig.Quantity)
	}

	nonce := time.Now().UTC().Unix() * 1000
	if issueConfig.FreeIssue {
		nonce = 0 // only the zero nonce is allowed for free issue
	}

	issues := make([]*transactionrecord.BitmarkIssue, issueConfig.Quantity)
	for i := 0; i < len(issues); i += 1 {
		issue, err := makeIssue(client.testnet, issueConfig, uint64(nonce)+uint64(i))
		if nil != err {
			return nil, err
		}
		if nil == issue {
			return nil, fault.ErrMakeIssueFailed
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

	// make response
	response := IssueReply{
		AssetId:        issues[0].AssetId, // Note: all issues are for the same asset
		IssueIds:       make([]merkle.Digest, len(issues)),
		PayId:          issuesReply.PayId,
		PayNonce:       issuesReply.PayNonce,
		Difficulty:     issuesReply.Difficulty,
		Payments:       issuesReply.Payments,
		SubmittedNonce: "",
	}

	if nil != issuesReply.Payments && len(issuesReply.Payments) > 0 {

		tpid, err := issuesReply.PayId.MarshalText()
		if nil != err {
			return nil, err
		}

		commands := make(map[string]string)
		for _, payment := range issuesReply.Payments {
			currency := payment[0].Currency
			commands[currency.String()] = paymentCommand(client.testnet, currency, string(tpid), payment)
		}
		response.Commands = commands

	} else {

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

		response.SubmittedNonce = proofArgs.Nonce
		response.ProofStatus = proofReply.Status

	}

	for i := 0; i < len(issuesReply.Issues); i++ {
		response.IssueIds[i] = issuesReply.Issues[i].TxId
	}

	return &response, nil
}

// build a properly signed issues
func makeIssue(testnet bool, issueConfig *IssueData, nonce uint64) (*transactionrecord.BitmarkIssue, error) {
	_, issue, err := internalMakeIssue(testnet, issueConfig, nonce, false)
	return issue, err
}

func internalMakeIssue(testnet bool, issueConfig *IssueData, nonce uint64, generateDigest bool) (*merkle.Digest, *transactionrecord.BitmarkIssue, error) {

	issuerAccount := issueConfig.Issuer.PrivateKey.Account()

	r := transactionrecord.BitmarkIssue{
		AssetId:   *issueConfig.AssetId,
		Owner:     issuerAccount,
		Nonce:     nonce,
		Signature: nil,
	}

	// pack without signature
	packed, err := r.Pack(issuerAccount)
	if nil == err {
		return nil, nil, fault.ErrMakeIssueFailed
	} else if fault.ErrInvalidSignature != err {
		return nil, nil, err
	}

	// manually sign the record and attach signature
	signature := ed25519.Sign(issueConfig.Issuer.PrivateKey.PrivateKeyBytes(), packed)
	r.Signature = signature[:]

	// check that signature is correct by packing again
	pkFull, err := r.Pack(issuerAccount)
	if nil != err {
		return nil, nil, err
	}
	if generateDigest {
		digest := merkle.NewDigest(pkFull)
		return &digest, &r, nil
	}
	return nil, &r, nil
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
				hps := float64(hashCount) / time.Since(start).Seconds() / 1.0e6
				fmt.Fprintf(handle, "%f MH/s: possible nonce: %x  with digest: %x\n", hps, nonceBuffer, digest)
			}
			hexDigest := hex.EncodeToString(digest[:])
			if hexDigest <= difficulty {
				return hex.EncodeToString(nonceBuffer)
			}
		}
	}
}
