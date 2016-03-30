// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/agl/ed25519"
	"github.com/bitmark-inc/bitmarkd/block"
	bFault "github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/rpc"
	"github.com/bitmark-inc/bitmarkd/transaction"
	"github.com/bitmark-inc/bitmark-cli/fault"
	"net"
	netrpc "net/rpc"
	"time"
)

// to hold a keypair for testing
type keyPair struct {
	publicKey  [32]byte
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
	testNet  bool
}

type issueData struct {
	issuer     keyPair
	assetIndex *transaction.AssetIndex
	quantity   int
}

type transferData struct {
	owner    keyPair
	newOwner keyPair
	txId     string
}

// a dummy signature to begin
var dummySignature transaction.Signature

// helper to make an address
func makeAddress(publicKey *[32]byte, testNet bool) *transaction.Address {
	return &transaction.Address{
		AddressInterface: &transaction.ED25519Address{
			Test:      testNet,
			PublicKey: publicKey,
		},
	}
}

// build a properly signed asset
func makeAsset(client *netrpc.Client, testNet bool, assetConfig assetData, verbose bool) (*transaction.AssetIndex, error) {

	registrantAddress := makeAddress(&assetConfig.registrant.publicKey, testNet)

	r := transaction.AssetData{
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
	signature := ed25519.Sign(&assetConfig.registrant.privateKey, packed)
	r.Signature = signature[:]

	// re-pack with correct signature
	if _, err = r.Pack(registrantAddress); nil != err {
		fmt.Printf("pack error: %v\n", err)
		return nil, fault.ErrMakeAssetFail
	}

	if verbose {
		fmt.Println("**** Create Asset ****")
		if err = printJson("Asset Request", r); nil != err {
			return nil, err
		}
	}

	var reply rpc.AssetRegisterReply
	if err := client.Call("Asset.Register", r, &reply); err != nil {
		fmt.Printf("Asset.Register error: %v\n", err)
		return nil, fault.ErrAssetRequestFail
	}

	if verbose {
		if err := printJson("Asset Reply", reply); nil != err {
			return nil, err
		}
	}

	return &reply.AssetIndex, nil
}

// build a properly signed issues
func makeIssue(testNet bool, issueConfig issueData, nonce uint64) *transaction.BitmarkIssue {

	issuerAddress := makeAddress(&issueConfig.issuer.publicKey, testNet)

	r := transaction.BitmarkIssue{
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
	signature := ed25519.Sign(&issueConfig.issuer.privateKey, packed)
	r.Signature = signature[:]

	// re-pack with correct signature
	if _, err = r.Pack(issuerAddress); nil != err {
		fmt.Printf("pack error: %v\n", err)
		return nil
	}
	return &r
}

type issueReply struct {
	AssetId        transaction.AssetIndex `json:"assetId"`
	IssueIds       []transaction.Link     `json:"issueIds"`
	PaymentAddress []block.MinerAddress   `json:"paymentAddress"`
	Err            string                 `json:"error,omitempty"`
}

func doIssues(client *netrpc.Client, testNet bool, issueConfig issueData, verbose bool) error {

	nonce := time.Now().UTC().Unix() * 1000
	issues := make([]*transaction.BitmarkIssue, issueConfig.quantity)
	for i := 0; i < len(issues); i += 1 {
		if issues[i] = makeIssue(testNet, issueConfig, uint64(nonce)+uint64(i)); nil == issues[i] {
			return fault.ErrMakeIssueFail
		}
	}

	if verbose {
		fmt.Println("**** Create Issue ****")
		if err := printJson("Issue Request", issues); nil != err {
			return err
		}
	}

	var reply []rpc.BitmarkIssueReply
	if err := client.Call("Bitmarks.Issue", issues, &reply); err != nil {
		fmt.Printf("Bitmark.Issue error: %v\n", err)
		return fault.ErrIssueRequestFail
	}

	if verbose {
		if err := printJson("Issue Reply", reply); nil != err {
			return err
		}
	} else { // make response
		response := issueReply{
			AssetId:        issues[0].AssetIndex,
			IssueIds:       make([]transaction.Link, len(issues)),
			PaymentAddress: reply[0].PaymentAddress,
			// make([]block.MinerAddress)
		}

		// remove duplicate payment address
		for i := 0; i < len(reply); i++ {
			response.IssueIds[i] = reply[i].TxId
			response.Err = reply[i].Err

			if i > 0 {
				for j := 0; j < len(reply[i].PaymentAddress); j++ {
					needAdd := true
					for k := 0; k < len(response.PaymentAddress); k++ {
						if reply[i].PaymentAddress[j].Address == response.PaymentAddress[k].Address {
							needAdd = false
							break
						}
					}
					if needAdd {
						response.PaymentAddress[len(response.PaymentAddress)] = reply[i].PaymentAddress[j]
					}
				}
			}
		}

		if err := printJson("", response); nil != err {
			return err
		}
	}
	return nil
}

func makeTransfer(testNet bool, txId string, owner keyPair, newOwner keyPair) *transaction.BitmarkTransfer {
	var link transaction.Link
	if err := link.UnmarshalText([]byte(txId)); nil != err {
		fmt.Printf("make txId to link fail: %s\n", err)
		return nil
	}

	newOwnerAddress := makeAddress(&newOwner.publicKey, testNet)
	r := transaction.BitmarkTransfer{
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
	ownerAddress := makeAddress(&owner.publicKey, testNet)
	r.Signature = signature[:]

	// re-pack with correct signature
	if _, err = r.Pack(ownerAddress); nil != err {
		fmt.Printf("pack error: %v\n", err)
		return nil
	}
	return &r
}

type transferReply struct {
	TransferId     transaction.Link     `json:"transferId"`
	PaymentAddress []block.MinerAddress `json:"paymentAddress"`
	Err            string               `json:"error,omitempty"`
}

func doTransfer(client *netrpc.Client, testNet bool, transferConfig transferData, verbose bool) error {
	transfer := makeTransfer(testNet, transferConfig.txId, transferConfig.owner, transferConfig.newOwner)
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

	if verbose {
		if err := printJson("Transfer Reply", reply); nil != err {
			return err
		}
	} else { // make response
		responses := transferReply{
			TransferId:     reply.TxId,
			PaymentAddress: reply.PaymentAddress,
			Err:            reply.Err,
		}

		if err := printJson("", responses); nil != err {
			return err
		}
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
