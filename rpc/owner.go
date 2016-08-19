// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

// import (
// 	"github.com/bitmark-inc/bitmarkd/fault"
// 	"github.com/bitmark-inc/bitmarkd/transaction"
// 	"github.com/bitmark-inc/logger"
// )

// // Owner
// // -------

// type Owner struct {
// 	log *logger.L
// }

// // Owner bitmarks
// // --------------

// type OwnerBitmarksArguments struct {
// 	Owner *transaction.Address `json:"owner"`        // base58
// 	Start uint64               `json:"start,string"` // first record number
// 	Count int                  `json:"count"`        // number of records
// }

// type OwnerBitmarksReply struct {
// 	Next uint64                    `json:"next,string"` // start value for the next call
// 	Data []transaction.Ownership   `json:"data"`        // list of bitmarks either issue or transfer
// 	Tx   map[string]BitmarksRecord `json:"tx"`          // table of tx records
// }

// // can be any of the transaction records
// type BitmarksRecord struct {
// 	Record string            `json:"record"`
// 	TxId   transaction.Link  `json:"txId"`
// 	State  transaction.State `json:"state"`
// 	Data   interface{}       `json:"data"`
// }

// func (owner *Owner) Bitmarks(arguments *OwnerBitmarksArguments, reply *OwnerBitmarksReply) error {
// 	log := owner.log
// 	log.Debugf("Owner.Bitmarks: %v", arguments)

// 	ownership, err := transaction.FetchOwnership(arguments.Owner, arguments.Start, arguments.Count)
// 	if nil != err {
// 		return err
// 	}

// 	// extract unique TxIds
// 	//   issues TxId == IssueTxId
// 	//   assets could be duplicates
// 	ids := make(map[transaction.Link]struct{})
// 	current := uint64(0)
// 	for _, r := range ownership {
// 		ids[r.TxId] = struct{}{}
// 		ids[r.IssueTxId] = struct{}{}
// 		ids[r.AssetTxId] = struct{}{}
// 		current = r.N
// 	}

// 	records := make(map[string]BitmarksRecord)

// 	for id := range ids {
// 		state, data, found := id.Read()
// 		if !found {
// 			return fault.ErrLinkNotFound
// 		}

// 		tx, err := data.Unpack()
// 		if nil != err {
// 			return err
// 		}

// 		record, ok := transaction.RecordName(tx)
// 		if !ok {
// 			return fault.ErrInvalidType
// 		}
// 		txId, err := id.MarshalText()
// 		if nil != err {
// 			return err
// 		}

// 		records[string(txId)] = BitmarksRecord{
// 			Record: record,
// 			TxId:   id,
// 			State:  state,
// 			Data:   tx,
// 		}
// 	}
// 	reply.Data = ownership
// 	reply.Tx = records

// 	// if no record were found the just return Next as zero
// 	// otherwise the next possible number
// 	if 0 == current {
// 		reply.Next = 0
// 	} else {
// 		reply.Next = current + 1
// 	}
// 	return nil
// }
