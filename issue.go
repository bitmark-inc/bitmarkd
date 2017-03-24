// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/bitmark-inc/bitmark-cli/fault"
	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/currency"
	bFault "github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/bitmarkd/rpc"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"golang.org/x/crypto/ed25519"
	"net"
	netrpc "net/rpc"
	"time"
)

// prefix for the payment command
// assumed format is: paymentCommand paymentNetwork='network' 'PaymentId' 'BTCaddress₁' 'SatoshiAmount₁' … 'BTCaddressN' 'SatoshiAmountN'
const (
	paymentCommand = "bitmark-pay --json"
	paymentNetwork = "--network="
)

type assetData struct {
	name        string
	metadata    string
	quantity    int
	registrant  *KeyPair
	fingerprint string
}

type bitmarkRPC struct {
	hostPort string
	network  string
}

type issueData struct {
	issuer     *KeyPair
	assetIndex *transactionrecord.AssetIndex
	quantity   int
}

type transferData struct {
	owner    *KeyPair
	newOwner *KeyPair
	txId     string
}

type receiptData struct {
	payId   string
	receipt string
}

type provenanceData struct {
	txId  string
	count int
}

// a dummy signature to begin
var dummySignature account.Signature

// helper to make an address
func makeAddress(keyPair *KeyPair, network string) *account.Account {

	testNet := true
	if network == "bitmark" {
		testNet = false
	}

	return &account.Account{
		AccountInterface: &account.ED25519Account{
			Test:      testNet,
			PublicKey: keyPair.PublicKey[:],
		},
	}
}

// helper to make a private key
func makePrivateKey(keyPair *KeyPair, network string) *account.PrivateKey {

	testNet := true
	if network == "bitmark" {
		testNet = false
	}

	return &account.PrivateKey{
		PrivateKeyInterface: &account.ED25519PrivateKey{
			Test:       testNet,
			PrivateKey: keyPair.PrivateKey[:],
		},
	}
}

// build a properly signed asset
func makeAsset(client *netrpc.Client, network string, assetConfig assetData, verbose bool) (*transactionrecord.AssetIndex, error) {

	assetIndex := (*transactionrecord.AssetIndex)(nil)

	getArgs := rpc.AssetGetArguments{
		Fingerprints: []string{assetConfig.fingerprint},
	}
	if verbose {
		if err := printJson("Asset Get Request", getArgs); nil != err {
			return nil, err
		}
	}

	var getReply rpc.AssetGetReply
	if err := client.Call("Assets.Get", &getArgs, &getReply); nil != err {
		return nil, err
	}

	if 1 != len(getReply.Assets) {
		return nil, fault.ErrAssetRequestFail
	}

	switch getReply.Assets[0].Record {
	case "AssetData":
		ar, ok := getReply.Assets[0].Data.(map[string]interface{})
		if !ok {
			return nil, fault.ErrAssetRequestFail
		}

		if ar["metadata"] != assetConfig.metadata {
			return nil, fault.ErrAssetRequestFail
		}
		if ar["name"] != assetConfig.name {
			return nil, fault.ErrAssetRequestFail
		}

		buffer, ok := getReply.Assets[0].AssetIndex.(string)
		if !ok {
			return nil, fault.ErrAssetRequestFail
		}
		var ai transactionrecord.AssetIndex
		err := ai.UnmarshalText([]byte(buffer))
		if nil != err {
			return nil, err
		}
		assetIndex = &ai

	default:
		if nil != getReply.Assets[0].Data {
			return nil, fault.ErrAssetRequestFail
		}
	}

	if verbose {
		if err := printJson("Asset Get Reply", getReply); nil != err {
			return nil, err
		}
	}

	if nil != assetIndex {
		return assetIndex, nil
	}

	registrantAddress := makeAddress(assetConfig.registrant, network)

	r := transactionrecord.AssetData{
		Name:        assetConfig.name,
		Fingerprint: assetConfig.fingerprint,
		Metadata:    assetConfig.metadata,
		Registrant:  registrantAddress,
		Signature:   dummySignature,
	}

	packed, err := r.Pack(registrantAddress)
	if bFault.ErrInvalidSignature != err {
		return nil, err
	}

	// manually sign the record and attach signature
	signature := ed25519.Sign(assetConfig.registrant.PrivateKey, packed)
	r.Signature = signature[:]

	// re-pack with correct signature
	if _, err = r.Pack(registrantAddress); nil != err {
		return nil, err
	}

	if verbose {
		if err := printJson("Asset Request", r); nil != err {
			return nil, err
		}
	}

	args := rpc.CreateArguments{
		Assets: []*transactionrecord.AssetData{&r},
		Issues: nil,
	}

	var reply rpc.CreateReply
	if err := client.Call("Bitmarks.Create", &args, &reply); nil != err {
		return nil, err
	}

	if verbose {
		if err := printJson("Asset Reply", reply); nil != err {
			return nil, err
		}
	}

	return reply.Assets[0].AssetIndex, nil
}

// build a properly signed issues
func makeIssue(network string, issueConfig issueData, nonce uint64) (*transactionrecord.BitmarkIssue, error) {

	issuerAddress := makeAddress(issueConfig.issuer, network)

	r := transactionrecord.BitmarkIssue{
		AssetIndex: *issueConfig.assetIndex,
		Owner:      issuerAddress,
		Nonce:      nonce,
		Signature:  dummySignature,
	}

	packed, err := r.Pack(issuerAddress)
	if bFault.ErrInvalidSignature != err {
		return nil, err
	}

	// manually sign the record and attach signature
	signature := ed25519.Sign(issueConfig.issuer.PrivateKey, packed)
	r.Signature = signature[:]

	// re-pack with correct signature
	if _, err = r.Pack(issuerAddress); nil != err {
		return nil, err
	}
	return &r, nil
}

// JSON data to output after asset/issue/proof completes
type issueReply struct {
	AssetId        transactionrecord.AssetIndex `json:"assetId"`
	IssueIds       []merkle.Digest              `json:"issueIds"`
	PayId          pay.PayId                    `json:"payId"`
	PayNonce       reservoir.PayNonce           `json:"payNonce"`
	Difficulty     string                       `json:"difficulty"`
	SubmittedNonce string                       `json:"submittedNonce"`
	ProofStatus    reservoir.TrackingStatus     `json:"proofStatus"`
}

func doIssues(client *netrpc.Client, network string, issueConfig issueData, verbose bool) error {

	nonce := time.Now().UTC().Unix() * 1000
	issues := make([]*transactionrecord.BitmarkIssue, issueConfig.quantity)
	for i := 0; i < len(issues); i += 1 {
		issue, err := makeIssue(network, issueConfig, uint64(nonce)+uint64(i))
		if nil != err {
			return err
		}
		if nil == issue {
			return fault.ErrMakeIssueFail
		}
		issues[i] = issue
	}

	if verbose {
		if err := printJson("Issue Request", issues); nil != err {
			return err
		}
	}

	issuesArgs := rpc.CreateArguments{
		Assets: nil,
		Issues: issues,
	}

	var issuesReply rpc.CreateReply
	if err := client.Call("Bitmarks.Create", issuesArgs, &issuesReply); err != nil {
		return err
	}

	if verbose {
		if err := printJson("Issue Reply", issuesReply); nil != err {
			return err
		}
	}

	// run proofer to generate local nonce
	localNonce := makeProof(issuesReply.PayId, issuesReply.PayNonce, issuesReply.Difficulty, verbose)
	proofArgs := rpc.ProofArguments{
		PayId: issuesReply.PayId,
		Nonce: localNonce,
	}

	if verbose {
		if err := printJson("Proof Request", proofArgs); nil != err {
			return err
		}
	}

	var proofReply rpc.ProofReply
	if err := client.Call("Bitmarks.Proof", &proofArgs, &proofReply); err != nil {
		return err
	}

	if verbose {
		if err := printJson("Proof Reply", proofReply); nil != err {
			return err
		}
	} else { // make response
		response := issueReply{
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

		if err := printJson("", response); nil != err {
			return err
		}
	}

	return nil
}

func makeTransfer(network string, txId string, owner *KeyPair, newOwner *KeyPair) (*transactionrecord.BitmarkTransfer, error) {
	var link merkle.Digest
	if err := link.UnmarshalText([]byte(txId)); nil != err {
		return nil, err
	}

	newOwnerAddress := makeAddress(newOwner, network)
	r := transactionrecord.BitmarkTransfer{
		Link:      link,
		Owner:     newOwnerAddress,
		Signature: dummySignature,
	}

	packed, err := r.Pack(newOwnerAddress)
	if bFault.ErrInvalidSignature != err {
		return nil, err
	}

	signature := ed25519.Sign(owner.PrivateKey, packed)
	ownerAddress := makeAddress(owner, network)
	r.Signature = signature[:]

	// re-pack with correct signature
	if _, err = r.Pack(ownerAddress); nil != err {
		return nil, err
	}
	return &r, nil
}

// JSON data to output after transfer completes
type transferReply struct {
	TransferId merkle.Digest                `json:"transferId"`
	PayId      pay.PayId                    `json:"payId"`
	Payments   []*transactionrecord.Payment `json:"payments"`
	Command    string                       `json:"command,omitempty"`
}

func doTransfer(client *netrpc.Client, network string, transferConfig transferData, verbose bool) error {
	transfer, err := makeTransfer(network, transferConfig.txId, transferConfig.owner, transferConfig.newOwner)
	if nil != err {
		return err
	}
	if nil == transfer {
		return fault.ErrMakeTransferFail
	}

	if verbose {
		if err := printJson("Transfer Request", transfer); nil != err {
			return err
		}
	}

	var reply rpc.BitmarkTransferReply
	if err := client.Call("Bitmark.Transfer", transfer, &reply); err != nil {
		return err
	}

	tpid, err := reply.PayId.MarshalText()
	if nil != err {
		return err
	}

	command := paymentCommand +
		" " + paymentNetwork + "'" + network +
		"' '" + string(tpid) + "'"
	for _, p := range reply.Payments {

		switch p.Currency {
		case currency.Bitcoin:
			command += fmt.Sprintf(" '%s' '%d'", p.Address, p.Amount)
		default:
			command += fmt.Sprintf(" 'UNKNOWN-%s' '%d'", p.Address, p.Amount)
		}
	}

	if verbose {
		if err := printJson("Transfer Reply", reply); nil != err {
			return err
		}
	} else { // make response
		responses := transferReply{
			TransferId: reply.TxId,
			PayId:      reply.PayId,
			Payments:   reply.Payments,
			Command:    command,
		}

		if err := printJson("", responses); nil != err {
			return err
		}
	}

	return nil
}

type receiptReply struct {
	Status reservoir.TrackingStatus `json:"status"`
}

func doReceipt(client *netrpc.Client, network string, receiptConfig receiptData, verbose bool) error {

	payArgs := rpc.PayArguments{
		Receipt: receiptConfig.receipt,
	}

	if err := payArgs.PayId.UnmarshalText([]byte(receiptConfig.payId)); nil != err {
		return err
	}

	if verbose {
		if err := printJson("Receipt Request", payArgs); nil != err {
			return err
		}
	}

	var reply rpc.PayReply
	if err := client.Call("Bitmarks.Pay", payArgs, &reply); err != nil {
		return err
	}

	if verbose {
		if err := printJson("Receipt Reply", reply); nil != err {
			return err
		}
	} else { // make response
		response := receiptReply{
			Status: reply.Status,
		}

		if err := printJson("", response); nil != err {
			return err
		}
	}

	return nil
}

func doProvenance(client *netrpc.Client, network string, provenanceConfig provenanceData, verbose bool) error {

	var txId merkle.Digest
	if err := txId.UnmarshalText([]byte(provenanceConfig.txId)); nil != err {
		return err
	}

	provenanceArgs := rpc.ProvenanceArguments{
		TxId:  txId,
		Count: provenanceConfig.count,
	}

	var reply rpc.ProvenanceReply
	if err := client.Call("Bitmark.Provenance", provenanceArgs, &reply); err != nil {
		return err
	}

	if verbose {
		err := printJson("Bitmark Provenance", reply)
		return err
	}

	if err := printJson("", reply); nil != err {
		return err
	}

	return nil
}

func getBitmarkInfo(client *netrpc.Client, verbose bool) error {
	var reply rpc.InfoReply
	if err := client.Call("Node.Info", rpc.InfoArguments{}, &reply); err != nil {
		return err
	}

	if verbose {
		err := printJson("Bitmarkd Info", reply)
		return err
	}

	if err := printJson("", reply); nil != err {
		return err
	}

	return nil
}

// connect to bitmarkd RPC
func connect(connect string) (net.Conn, error) {

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	conn, err := tls.Dial("tcp", connect, tlsConfig)
	if nil != err {
		return nil, err
	}

	return conn, nil
}

func printJson(title string, message interface{}) error {
	if b, err := json.MarshalIndent(message, "", "  "); nil != err {
		return err
	} else {
		if "" == title {
			fmt.Printf("%s\n", b)
		} else {
			fmt.Printf("%s:\n%s\n", title, b)
		}
	}

	return nil
}
