// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/agl/ed25519"
	"github.com/bitmark-inc/bitmark-cli/fault"
	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/currency"
	bFault "github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/payment"
	"github.com/bitmark-inc/bitmarkd/rpc"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"net"
	netrpc "net/rpc"
	"time"
)

// to hold a keypair for testing
type keyPair struct {
	publicKey  []byte
	privateKey [64]byte
}

type assetData struct {
	name        string
	description string
	quantity    int
	registrant  keyPair
	fingerprint string
}

type bitmarkRPC struct {
	hostPort string
	network  string
}

type issueData struct {
	issuer     keyPair
	assetIndex *transactionrecord.AssetIndex
	quantity   int
}

type transferData struct {
	owner    keyPair
	newOwner keyPair
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
func makeAddress(publicKey []byte, testNet bool) *account.Account {
	return &account.Account{
		AccountInterface: &account.ED25519Account{
			Test:      testNet,
			PublicKey: &tmpPublicKey,
		},
	}
}

// build a properly signed asset
func makeAsset(client *netrpc.Client, testNet bool, assetConfig assetData, verbose bool) (*transactionrecord.AssetIndex, error) {

	assetIndex := (*transactionrecord.AssetIndex)(nil)

	getArgs := rpc.AssetGetArguments{
		Fingerprints: []string{assetConfig.fingerprint},
	}
	if verbose {
		fmt.Println("**** Get Asset ****")
		if err := printJson("Asset Get Request", getArgs); nil != err {
			return nil, err
		}
	}

	var getReply rpc.AssetGetReply
	if err := client.Call("Assets.Get", &getArgs, &getReply); nil != err {
		fmt.Printf("Asset get error: %v\n", err)
		return nil, fault.ErrAssetRequestFail
	}

	if 1 != len(getReply.Assets) {
		fmt.Printf("Asset get returned incorrect data\n")
		return nil, fault.ErrAssetRequestFail
	}

	switch getReply.Assets[0].Record {
	case "AssetData":
		ar, ok := getReply.Assets[0].Data.(map[string]interface{})
		if !ok {
			fmt.Printf("Asset get returned no data\n")
			return nil, fault.ErrAssetRequestFail
		}

		if ar["description"] != assetConfig.description {
			fmt.Printf("Asset description mismatch: actual: %q  expected: %q:\n", ar["description"], assetConfig.description)
			return nil, fault.ErrAssetRequestFail
		}
		if ar["name"] != assetConfig.name {
			fmt.Printf("Asset name mismatch: actual: %q  expected: %q:\n", ar["name"], assetConfig.name)
			return nil, fault.ErrAssetRequestFail
		}

		buffer, ok := getReply.Assets[0].AssetIndex.(string)
		if !ok {
			fmt.Printf("Asset get returned no asset index\n")
			return nil, fault.ErrAssetRequestFail
		}
		var ai transactionrecord.AssetIndex
		err := ai.UnmarshalText([]byte(buffer))
		if nil != err {
			fmt.Printf("Asset Index conversion error: %v\n", err)
			return nil, err
		}
		assetIndex = &ai

	default:
		if nil != getReply.Assets[0].Data {
			fmt.Printf("Asset get returned non asset: %q\n", getReply.Assets[0].Record)
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

	registrantAddress := makeAddress(assetConfig.registrant.publicKey, testNet)

	r := transactionrecord.AssetData{
		Description: assetConfig.description,
		Name:        assetConfig.name,
		Fingerprint: assetConfig.fingerprint,
		Registrant:  registrantAddress,
		Signature:   dummySignature,
	}

	packed, err := r.Pack(registrantAddress)
	if bFault.ErrInvalidSignature != err {
		fmt.Printf("pack error: %v\n", err)
		return nil, fault.ErrMakeAssetFail
	}

	// manually sign the record and attach signature
	signature := ed25519.Sign(assetConfig.registrant.privateKey, packed)
	r.Signature = signature[:]

	// re-pack with correct signature
	if _, err = r.Pack(registrantAddress); nil != err {
		fmt.Printf("pack error: %v\n", err)
		return nil, fault.ErrMakeAssetFail
	}

	if verbose {
		fmt.Println("**** Create Asset ****")
		if err := printJson("Asset Request", r); nil != err {
			return nil, err
		}
	}

	args := rpc.CreateArguments{
		Assets: []transactionrecord.AssetData{r},
		Issues: nil,
	}

	var reply rpc.CreateReply
	if err := client.Call("Bitmarks.Create", &args, &reply); nil != err {
		fmt.Printf("Asset registration error: %v\n", err)
		return nil, fault.ErrAssetRequestFail
	}

	if verbose {
		if err := printJson("Asset Reply", reply); nil != err {
			return nil, err
		}
	}

	return reply.Assets[0].AssetIndex, nil
}

// build a properly signed issues
func makeIssue(testNet bool, issueConfig issueData, nonce uint64) *transactionrecord.BitmarkIssue {

	issuerAddress := makeAddress(issueConfig.issuer.publicKey, testNet)

	r := transactionrecord.BitmarkIssue{
		AssetIndex: *issueConfig.assetIndex,
		Owner:      issuerAddress,
		Nonce:      nonce,
		Signature:  dummySignature,
	}

	packed, err := r.Pack(issuerAddress)
	if bFault.ErrInvalidSignature != err {
		fmt.Printf("pack error: %v\n", err)
		return nil
	}

	// manually sign the record and attach signature
	signature := ed25519.Sign(issueConfig.issuer.privateKey, packed)
	r.Signature = signature[:]

	// re-pack with correct signature
	if _, err = r.Pack(issuerAddress); nil != err {
		fmt.Printf("pack error: %v\n", err)
		return nil
	}
	return &r
}

// JSON data to output after asset/issue/proof completes
type issueReply struct {
	AssetId        transactionrecord.AssetIndex `json:"assetId"`
	IssueIds       []merkle.Digest              `json:"issueIds"`
	PayId          payment.PayId                `json:"payId"`
	PayNonce       payment.PayNonce             `json:"payNonce"`
	Difficulty     string                       `json:"difficulty"`
	SubmittedNonce string                       `json:"submittedNonce"`
	ProofStatus    payment.TrackingStatus       `json:"proofStatus"`
}

func doIssues(client *netrpc.Client, network string, issueConfig issueData, verbose bool) error {

	nonce := time.Now().UTC().Unix() * 1000
	issues := make([]transactionrecord.BitmarkIssue, issueConfig.quantity)
	for i := 0; i < len(issues); i += 1 {
		issue := makeIssue(testNet, issueConfig, uint64(nonce)+uint64(i))
		if nil == issue {
			return fault.ErrMakeIssueFail
		}
		issues[i] = *issue
	}

	if verbose {
		fmt.Println("**** Create Issue ****")
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
		fmt.Printf("Bitmark.Create Issue error: %v\n", err)
		return fault.ErrIssueRequestFail
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
		fmt.Println("**** Send Proof ****")
		if err := printJson("Proof Request", proofArgs); nil != err {
			return err
		}
	}

	var proofReply rpc.ProofReply
	if err := client.Call("Bitmarks.Proof", &proofArgs, &proofReply); err != nil {
		fmt.Printf("Bitmarks.Proof error: %v\n", err)
		return fault.ErrIssueRequestFail
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

		if err := printJson("Issue Response", response); nil != err {
			return err
		}
	}

	return nil
}

func makeTransfer(testNet bool, txId string, owner keyPair, newOwner keyPair) *transactionrecord.BitmarkTransfer {
	var link merkle.Digest
	if err := link.UnmarshalText([]byte(txId)); nil != err {
		fmt.Printf("make txId to link fail: %s\n", err)
		return nil
	}

	newOwnerAddress := makeAddress(newOwner.publicKey, testNet)
	r := transactionrecord.BitmarkTransfer{
		Link:      link,
		Owner:     newOwnerAddress,
		Signature: dummySignature,
	}

	packed, err := r.Pack(newOwnerAddress)
	if bFault.ErrInvalidSignature != err {
		fmt.Printf("pack error: %v\n", err)
		return nil
	}

	signature := ed25519.Sign(&owner.privateKey, packed)
	ownerAddress := makeAddress(owner.publicKey, testNet)
	r.Signature = signature[:]

	// re-pack with correct signature
	if _, err = r.Pack(ownerAddress); nil != err {
		fmt.Printf("pack error: %v\n", err)
		return nil
	}
	return &r
}

// JSON data to output after transfer completes
type transferReply struct {
	TransferId merkle.Digest                `json:"transferId"`
	PayId      payment.PayId                `json:"payId"`
	Payments   []*transactionrecord.Payment `json:"payments"`
	Command    string                       `json:"command,omitempty"`
}

func doTransfer(client *netrpc.Client, network string, transferConfig transferData, verbose bool) error {
	transfer := makeTransfer(network, transferConfig.txId, transferConfig.owner, transferConfig.newOwner)
	if nil == transfer {
		return fault.ErrMakeTransferFail
	}

	if verbose {
		fmt.Println("**** Create Transfer ****")
		if err := printJson("Transfer Request", transfer); nil != err {
			return err
		}
	}

	var reply rpc.BitmarkTransferReply
	if err := client.Call("Bitmark.Transfer", transfer, &reply); err != nil {
		fmt.Printf("Bitmark.Transfer error: %v\n", err)
		return fault.ErrTransferRequestFail
	}

	tpid, err := reply.PayId.MarshalText()
	if nil != err {
		fmt.Printf("returned pay id error: %v\n", err)
		return fault.ErrTransferRequestFail
	}

	command := "make-payment --json '" + string(tpid) + "'"
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
	Status payment.TrackingStatus `json:"status"`
}

func doReceipt(client *netrpc.Client, testNet bool, receiptConfig receiptData, verbose bool) error {

	payArgs := rpc.PayArguments{
		Receipt: receiptConfig.receipt,
	}

	if err := payArgs.PayId.UnmarshalText([]byte(receiptConfig.payId)); nil != err {
		fmt.Printf("unmarshal pay id error: %v\n", err)
		return fault.ErrReceiptRequestFail
	}

	if verbose {
		fmt.Println("**** Create Receipt ****")
		if err := printJson("Receipt Request", payArgs); nil != err {
			return err
		}
	}

	var reply rpc.PayReply
	if err := client.Call("Bitmarks.Pay", payArgs, &reply); err != nil {
		fmt.Printf("Bitmarks.Pay error: %v\n", err)
		return fault.ErrReceiptRequestFail
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

func doProvenance(client *netrpc.Client, testNet bool, provenanceConfig provenanceData, verbose bool) error {

	var txId merkle.Digest
	if err := txId.UnmarshalText([]byte(provenanceConfig.txId)); nil != err {
		fmt.Printf("make txId to link fail: %s\n", err)
		return fault.ErrProvenanceRequestFail
	}

	provenanceArgs := rpc.ProvenanceArguments{
		TxId:  txId,
		Count: provenanceConfig.count,
	}

	var reply rpc.ProvenanceReply
	if err := client.Call("Bitmark.Provenance", provenanceArgs, &reply); err != nil {
		fmt.Printf("Bitmark.Provenance error: %v\n", err)
		return fault.ErrProvenanceRequestFail
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

func getInfo(client *netrpc.Client, verbose bool) error {
	var reply rpc.InfoReply
	if err := client.Call("Node.Info", rpc.InfoArguments{}, &reply); err != nil {
		fmt.Printf("Node.Info error: %v\n", err)
		return fault.ErrNodeInfoRequestFail
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
		fmt.Printf("json error: %v\n", err)
		return fault.ErrJsonParseFail
	} else {
		if "" == title {
			fmt.Printf("%s\n", b)
		} else {
			fmt.Printf("%s:\n%s\n", title, b)
		}
	}

	return nil
}
