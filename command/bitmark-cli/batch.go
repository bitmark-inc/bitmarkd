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

type progressType bool

func (p progressType) Printf(format string, arguments ...interface{}) {
	if p {
		fmt.Printf(format, arguments...)
	}
}

func batch(rpcConfig bitmarkRPC, assetConfig assetData, outputFilename string, verbose bool) error {

	progress := progressType(!verbose)

	conn, err := connect(rpcConfig.hostPort)
	if nil != err {
		return err
	}
	defer conn.Close()

	// create a client
	client := jsonrpc.NewClient(conn)
	defer client.Close()

	// make asset
	progress.Printf("make asset\n")
	assetIndex, err := makeAsset(client, rpcConfig.network, assetConfig, verbose)
	if nil != err {
		return err
	}

	// make Issues
	progress.Printf("make issues\n")
	issueConfig := issueData{
		issuer:     assetConfig.registrant,
		assetIndex: assetIndex,
		quantity:   assetConfig.quantity,
	}
	issueResult, err := doIssues(client, rpcConfig.network, issueConfig, verbose)
	if nil != err {
		return err
	}
	progress.Printf("issues completed\n")
	printJson("Issue Result", issueResult, verbose)

	progress.Printf("waiting for issues to be confirmed\n")
loop:
	for {
		progress.Printf(".")
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
			progress.Printf("\n")
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
		progress.Printf("create account: %d\n", i)

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

		progress.Printf("transfer a bitmark to account: %d\n", i)
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
	progress.Printf("saving data\n")
	if verbose {
		fmt.Printf("Result:\n")
	}
	if "" == outputFilename || "-" == outputFilename {
		printJson("", result)
	} else {
		printJsonToFile(outputFilename, result)
	}
	return nil
}
