// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpccalls

import (
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/rpc"
)

type ProvenanceData struct {
	TxId  string
	Count int
}

func (client *Client) GetProvenance(provenanceConfig *ProvenanceData) (*rpc.ProvenanceReply, error) {

	var txId merkle.Digest
	if err := txId.UnmarshalText([]byte(provenanceConfig.TxId)); nil != err {
		return nil, err
	}

	provenanceArgs := rpc.ProvenanceArguments{
		TxId:  txId,
		Count: provenanceConfig.Count,
	}

	client.printJson("Provenance Request", provenanceArgs)

	var reply rpc.ProvenanceReply
	err := client.client.Call("Bitmark.Provenance", provenanceArgs, &reply)
	if nil != err {
		return nil, err
	}

	client.printJson("Provenance Reply", reply)

	return &reply, nil
}
