// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"github.com/bitmark-inc/bitmarkd/rpc"
	"net/rpc/jsonrpc"
	"time"
)

func batch(rpcConfig bitmarkRPC, assetConfig assetData, verbose bool) error {

	conn, err := connect(rpcConfig.hostPort)
	if nil != err {
		return err
	}
	defer conn.Close()

	// create a client
	client := jsonrpc.NewClient(conn)
	defer client.Close()

	// make asset
	assetIndex, err := makeAsset(client, rpcConfig.network, assetConfig, verbose)
	if nil != err {
		return err
	}

	// make Issues
	issueConfig := issueData{
		issuer:     assetConfig.registrant,
		assetIndex: assetIndex,
		quantity:   assetConfig.quantity,
	}
	issueResult, err := doIssues(client, rpcConfig.network, issueConfig, verbose)
	if nil != err {
		return err
	}
	printJson("Issue Result", issueResult, verbose)

loop:
	for {
		confirmed := true
		for _, issueId := range issueResult.IssueIds {

			statusArgs := rpc.TransactionArguments{
				TxId: issueId,
			}
			printJson("Status Request", statusArgs, verbose)

			var reply rpc.TransactionStatusReply
			if err := client.Call("Transaction.Status", statusArgs, &reply); err != nil {
				return err
			}

			printJson("Status Reply", reply, verbose)

			if "Confirmed" != reply.Status {
				confirmed = false
			}
		}
		if confirmed {
			break loop
		}
		time.Sleep(5 * time.Second)
	}

	type accountAndTransfer struct {
		Account  *RawKeyPair    `json:"account"`
		Transfer *transferReply `json:"transfer"`
	}

	accounts := make([]accountAndTransfer, len(issueResult.IssueIds))

	for i, issueId := range issueResult.IssueIds {

		rawKeyPair, newOwnerKeyPair, err := makeRawKeyPair("bitmark" != rpcConfig.network)
		if nil != err {
			return err
		}
		printJson("New Key Pair", rawKeyPair, verbose)

		transferConfig := transferData{
			owner:    assetConfig.registrant,
			newOwner: newOwnerKeyPair,
			txId:     issueId,
		}

		transferResult, err := doTransfer(client, rpcConfig.network, transferConfig, verbose)
		if nil != err {
			return err
		}

		printJson("Transfer Result", transferResult, verbose)
		accounts[i].Account = rawKeyPair
		accounts[i].Transfer = transferResult

	}

	type finalResult struct {
		Issues *issueReply          `json:"issues"`
		Items  []accountAndTransfer `json:"items"`
	}
	result := finalResult{
		Issues: issueResult,
		Items:  accounts,
	}
	if verbose {
		fmt.Printf("Result:\n")
	}
	printJson("", result)

	return nil
}
