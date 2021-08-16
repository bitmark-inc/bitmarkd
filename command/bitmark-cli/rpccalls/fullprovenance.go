// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2021 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpccalls

import (
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/rpc/bitmark"
)

// FullProvenanceData - data for a full provenance request
type FullProvenanceData struct {
	BitmarkId  string
	Identities map[string]string
}

// FullProvenanceReply - list of transactions in the full provenance chain
type FullProvenanceReply struct {
	Data []fullProvenanceItem `json:"data"`
}

// fullProvenanceItem - transaction record in full provenance chain
type fullProvenanceItem struct {
	bitmark.FullProvenanceRecord
	Identity string `json:"_IDENTITY,omitempty"`
}

// GetFullProvenance - obtain the full provenance chain from a specific bitmark id
func (client *Client) GetFullProvenance(fullProvenanceConfig *FullProvenanceData) (*FullProvenanceReply, error) {

	var bitmarkId merkle.Digest
	if err := bitmarkId.UnmarshalText([]byte(fullProvenanceConfig.BitmarkId)); nil != err {
		return nil, err
	}

	fullProvenanceArgs := bitmark.FullProvenanceArguments{
		BitmarkId: bitmarkId,
	}

	client.printJson("Full Provenance Request", fullProvenanceArgs)

	var reply bitmark.FullProvenanceReply
	err := client.client.Call("Bitmark.FullProvenance", fullProvenanceArgs, &reply)
	if nil != err {
		return nil, err
	}

	client.printJson("Full Provenance Reply", reply)

	r := &FullProvenanceReply{
		Data: make([]fullProvenanceItem, len(reply.Data)),
	}

	for i, d := range reply.Data {
		r.Data[i].FullProvenanceRecord = d

		m := d.Data.(map[string]interface{})
		owner := m["owner"]
		if nil == owner {
			owner = m["registrant"]
		}
		if s, ok := owner.(string); ok {
			r.Data[i].Identity = fullProvenanceConfig.Identities[s]
		}
	}

	return r, nil
}
