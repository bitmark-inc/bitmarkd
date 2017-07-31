// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"crypto/tls"
	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/keypair"
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

var (
	ErrMakeIssueFail    = fault.ProcessError("make issue failed")
	ErrAssetRequestFail = fault.ProcessError("send asset request failed")
	ErrMakeTransferFail = fault.ProcessError("make transfer failed")
)

type assetData struct {
	name        string
	metadata    string
	quantity    int
	registrant  *keypair.KeyPair
	fingerprint string
}

type bitmarkRPC struct {
	hostPort string
	network  string
}

type issueData struct {
	issuer     *keypair.KeyPair
	assetIndex *transactionrecord.AssetIndex
	quantity   int
}

type transferData struct {
	owner    *keypair.KeyPair
	newOwner *keypair.KeyPair
	txId     merkle.Digest
}

type receiptData struct {
	payId   string
	receipt string
}

type provenanceData struct {
	txId  string
	count int
}

type transactionStatusData struct {
	txId string
}

// a dummy signature to begin
var dummySignature account.Signature

// helper to make an address
func makeAddress(keyPair *keypair.KeyPair, network string) *account.Account {

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
func makePrivateKey(keyPair *keypair.KeyPair, network string) *account.PrivateKey {

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

	printJson("Asset Get Request", getArgs, verbose)

	var getReply rpc.AssetGetReply
	if err := client.Call("Assets.Get", &getArgs, &getReply); nil != err {
		return nil, err
	}

	if 1 != len(getReply.Assets) {
		return nil, ErrAssetRequestFail
	}

	switch getReply.Assets[0].Record {
	case "AssetData":
		ar, ok := getReply.Assets[0].Data.(map[string]interface{})
		if !ok {
			return nil, ErrAssetRequestFail
		}

		if ar["metadata"] != assetConfig.metadata {
			return nil, ErrAssetRequestFail
		}
		if ar["name"] != assetConfig.name {
			return nil, ErrAssetRequestFail
		}

		buffer, ok := getReply.Assets[0].AssetIndex.(string)
		if !ok {
			return nil, ErrAssetRequestFail
		}
		var ai transactionrecord.AssetIndex
		err := ai.UnmarshalText([]byte(buffer))
		if nil != err {
			return nil, err
		}
		assetIndex = &ai

	default:
		if nil != getReply.Assets[0].Data {
			return nil, ErrAssetRequestFail
		}
	}

	printJson("Asset Get Reply", getReply, verbose)

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
	if fault.ErrInvalidSignature != err {
		return nil, err
	}

	// manually sign the record and attach signature
	signature := ed25519.Sign(assetConfig.registrant.PrivateKey, packed)
	r.Signature = signature[:]

	// re-pack with correct signature
	if _, err = r.Pack(registrantAddress); nil != err {
		return nil, err
	}

	printJson("Asset Request", r, verbose)

	args := rpc.CreateArguments{
		Assets: []*transactionrecord.AssetData{&r},
		Issues: nil,
	}

	var reply rpc.CreateReply
	if err := client.Call("Bitmarks.Create", &args, &reply); nil != err {
		return nil, err
	}

	printJson("Asset Reply", reply, verbose)

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
	if fault.ErrInvalidSignature != err {
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

func doIssues(client *netrpc.Client, network string, issueConfig issueData, verbose bool) (*issueReply, error) {

	nonce := time.Now().UTC().Unix() * 1000
	issues := make([]*transactionrecord.BitmarkIssue, issueConfig.quantity)
	for i := 0; i < len(issues); i += 1 {
		issue, err := makeIssue(network, issueConfig, uint64(nonce)+uint64(i))
		if nil != err {
			return nil, err
		}
		if nil == issue {
			return nil, ErrMakeIssueFail
		}
		issues[i] = issue
	}

	printJson("Issue Request", issues, verbose)

	issuesArgs := rpc.CreateArguments{
		Assets: nil,
		Issues: issues,
	}

	var issuesReply rpc.CreateReply
	if err := client.Call("Bitmarks.Create", issuesArgs, &issuesReply); err != nil {
		return nil, err
	}

	printJson("Issue Reply", issuesReply, verbose)

	// run proofer to generate local nonce
	localNonce := makeProof(issuesReply.PayId, issuesReply.PayNonce, issuesReply.Difficulty, verbose)
	proofArgs := rpc.ProofArguments{
		PayId: issuesReply.PayId,
		Nonce: localNonce,
	}

	printJson("Proof Request", proofArgs, verbose)

	var proofReply rpc.ProofReply
	if err := client.Call("Bitmarks.Proof", &proofArgs, &proofReply); err != nil {
		return nil, err
	}

	printJson("Proof Reply", proofReply, verbose)

	// make response
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

	return &response, nil
}

func makeTransfer(network string, link merkle.Digest, owner *keypair.KeyPair, newOwner *keypair.KeyPair) (*transactionrecord.BitmarkTransfer, error) {

	newOwnerAddress := makeAddress(newOwner, network)
	r := transactionrecord.BitmarkTransfer{
		Link:      link,
		Owner:     newOwnerAddress,
		Signature: dummySignature,
	}

	packed, err := r.Pack(newOwnerAddress)
	if fault.ErrInvalidSignature != err {
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
	TransferId merkle.Digest                                   `json:"transferId"`
	PayId      pay.PayId                                       `json:"payId"`
	Payments   map[string]transactionrecord.PaymentAlternative `json:"payments"`
	Commands   map[string]string                               `json:"commands,omitempty"`
}

func doTransfer(client *netrpc.Client, network string, transferConfig transferData, verbose bool) (*transferReply, error) {
	transfer, err := makeTransfer(network, transferConfig.txId, transferConfig.owner, transferConfig.newOwner)
	if nil != err {
		return nil, err
	}
	if nil == transfer {
		return nil, ErrMakeTransferFail
	}

	printJson("Transfer Request", transfer, verbose)

	var reply rpc.BitmarkTransferReply
	if err := client.Call("Bitmark.Transfer", transfer, &reply); err != nil {
		return nil, err
	}

	tpid, err := reply.PayId.MarshalText()
	if nil != err {
		return nil, err
	}

	commands := make(map[string]string)
	for _, payment := range reply.Payments {
		currency := payment[0].Currency
		commands[currency.String()] = paymentCommand(network, currency, string(tpid), payment)
	}

	printJson("Transfer Reply", reply, verbose)

	// make response
	response := transferReply{
		TransferId: reply.TxId,
		PayId:      reply.PayId,
		Payments:   reply.Payments,
		Commands:   commands,
	}

	return &response, nil
}

func doProvenance(client *netrpc.Client, network string, provenanceConfig provenanceData, verbose bool) (*rpc.ProvenanceReply, error) {

	var txId merkle.Digest
	if err := txId.UnmarshalText([]byte(provenanceConfig.txId)); nil != err {
		return nil, err
	}

	provenanceArgs := rpc.ProvenanceArguments{
		TxId:  txId,
		Count: provenanceConfig.count,
	}

	printJson("Provenance Request", provenanceArgs, verbose)

	var reply rpc.ProvenanceReply
	if err := client.Call("Bitmark.Provenance", provenanceArgs, &reply); err != nil {
		return nil, err
	}

	printJson("Provenance Reply", reply, verbose)

	return &reply, nil
}

func doTransactionStatus(client *netrpc.Client, network string, statusConfig transactionStatusData, verbose bool) (*rpc.TransactionStatusReply, error) {

	var txId merkle.Digest
	if err := txId.UnmarshalText([]byte(statusConfig.txId)); nil != err {
		return nil, err
	}

	statusArgs := rpc.TransactionArguments{
		TxId: txId,
	}

	printJson("Status Request", statusArgs, verbose)

	var reply rpc.TransactionStatusReply
	if err := client.Call("Transaction.Status", statusArgs, &reply); err != nil {
		return nil, err
	}

	printJson("Status Reply", reply, verbose)

	return &reply, nil
}

func getBitmarkInfo(client *netrpc.Client, verbose bool) (*rpc.InfoReply, error) {
	var reply rpc.InfoReply
	if err := client.Call("Node.Info", rpc.InfoArguments{}, &reply); err != nil {
		return nil, err
	}

	printJson("Info Reply", reply, verbose)

	return &reply, nil
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

// converts atx id string to digest
func txIdFromString(txId string) (merkle.Digest, error) {
	var link merkle.Digest
	if err := link.UnmarshalText([]byte(txId)); nil != err {
		return merkle.Digest{}, err
	}
	return link, nil
}
