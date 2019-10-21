// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package reservoir

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"

	"github.com/bitmark-inc/bitmarkd/storage"

	"github.com/bitmark-inc/bitmarkd/asset"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
)

type tagType byte

// record types in cache file
const (
	taggedBOF         tagType = iota
	taggedEOF         tagType = iota
	taggedTransaction tagType = iota
	taggedProof       tagType = iota
)

// the BOF tag to chec file version
// exact match is required
var bofData = []byte("bitmark-cache v1.0")

// LoadFromFile - load transactions from file
// called later when system is able to handle the tx and proofs
func LoadFromFile(assetHandle storage.Handle, blockOwnerPaymentHandle storage.Handle) error {
	Disable()
	defer Enable()

	log := globalData.log

	log.Info("starting…")

	f, err := os.Open(globalData.filename)
	if nil != err {
		return err
	}
	defer f.Close()

	// must have BOF record first
	tag, packed, err := readRecord(f)
	if nil != err {
		return err
	}

	if taggedBOF != tag {
		return fmt.Errorf("expected BOF: %d but read: %d", taggedBOF, tag)
	}

	if !bytes.Equal(bofData, packed) {
		return fmt.Errorf("expected BOF: %q but read: %q", bofData, packed)
	}

	log.Infof("restore from file: %s", globalData.filename)

restore_loop:
	for {
		tag, packed, err := readRecord(f)
		if nil != err {
			return err
		}
		switch tag {

		case taggedEOF:
			break restore_loop

		case taggedTransaction:
			unpacked, _, err := packed.Unpack(mode.IsTesting())
			if nil != err {
				log.Errorf("unable to unpack asset: %s", err)
				continue restore_loop
			}
			switch tx := unpacked.(type) {

			case *transactionrecord.AssetData:
				_, _, err := asset.Cache(tx)
				if nil != err {
					log.Errorf("fail to cache asset: %s", err)
				}

			case *transactionrecord.BitmarkIssue:
				packedIssues := packed
				issues := make([]*transactionrecord.BitmarkIssue, 0, 100)

				for len(packedIssues) > 0 {
					transaction, n, err := packedIssues.Unpack(mode.IsTesting())
					if nil != err {
						log.Errorf("unable to unpack issue: %s", err)
						continue restore_loop
					}

					if issue, ok := transaction.(*transactionrecord.BitmarkIssue); ok {
						issues = append(issues, issue)
					} else {
						log.Errorf("issue block contains non-issue: %+v", transaction)
						continue restore_loop
					}
					packedIssues = packedIssues[n:]
				}

				_, _, err := StoreIssues(issues, assetHandle, blockOwnerPaymentHandle)
				if nil != err {
					log.Errorf("fail to store issue: %s", err)
				}

			case *transactionrecord.BitmarkTransferUnratified,
				*transactionrecord.BitmarkTransferCountersigned:
				tr := tx.(transactionrecord.BitmarkTransfer)
				_, _, err := StoreTransfer(tr)
				if nil != err {
					log.Errorf("fail to store transfer: %s", err)
				}

			default:
				log.Errorf("read invalid transaction: %+v", tx)
				return fmt.Errorf("read invalid transaction")
			}

		case taggedProof:
			var payId pay.PayId
			pn := len(payId)
			if len(packed) <= pn {
				log.Errorf("unable to unpack proof: record too short: %d  expected > %d", len(packed), pn)
				continue restore_loop
			}
			copy(payId[:], packed[:pn])
			nonce := packed[pn:]
			TryProof(payId, nonce)

		default:
			log.Errorf("read invalid tag: 0x%02x", tag)
			return fmt.Errorf("read invalid tag: 0x%02x", tag)
		}
	}
	log.Info("restore completed")
	return nil
}

// save transactions to file
func saveToFile() error {
	globalData.Lock()
	defer globalData.Unlock()

	log := globalData.log

	if !globalData.initialised {
		log.Error("save when not initialised")
		return fault.NotInitialised
	}

	log.Info("saving…")

	f, err := os.OpenFile(globalData.filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if nil != err {
		return err
	}
	defer f.Close()

	// write beginning of file marker
	err = writeRecord(f, taggedBOF, bofData)
	if nil != err {
		return err
	}

	// all assets at start of file
	err = backupAssets(f)
	if nil != err {
		return err
	}

	// verified

	for _, item := range globalData.verifiedTransactions {
		err := writeRecord(f, taggedTransaction, item.packed)
		if nil != err {
			return err
		}
	}
	for _, item := range globalData.verifiedFreeIssues {
		err := writeBlock(f, taggedTransaction, item.txs)
		if nil != err {
			return err
		}
		err = writeRecord(f, taggedProof, packProof(item.payId, item.nonce))
		if nil != err {
			return err
		}
	}
	for _, item := range globalData.verifiedPaidIssues {
		err := writeBlock(f, taggedTransaction, item.txs)
		if nil != err {
			return err
		}
	}

	// pending

	for _, item := range globalData.pendingTransactions {
		err := writeRecord(f, taggedTransaction, item.tx.packed)
		if nil != err {
			return err
		}
	}
	for _, item := range globalData.pendingFreeIssues {
		err := writeBlock(f, taggedTransaction, item.txs)
		if nil != err {
			return err
		}
		err = writeRecord(f, taggedProof, packProof(item.payId, item.nonce))
		if nil != err {
			return err
		}
	}
	for _, item := range globalData.pendingPaidIssues {
		err := writeBlock(f, taggedTransaction, item.txs)
		if nil != err {
			return err
		}
	}

	// end the file
	err = writeRecord(f, taggedEOF, []byte("EOF"))
	if nil != err {
		return err
	}

	log.Info("save completed")
	return nil
}

func backupAssets(f *os.File) error {
	allAssets := make(map[transactionrecord.AssetIdentifier]struct{})

	// verified

	for _, item := range globalData.verifiedFreeIssues {
		for _, tx := range item.txs {
			if tx, ok := tx.transaction.(*transactionrecord.BitmarkIssue); ok {
				allAssets[tx.AssetId] = struct{}{}
			}
		}
	}
	for _, item := range globalData.verifiedPaidIssues {
		for _, tx := range item.txs {
			if tx, ok := tx.transaction.(*transactionrecord.BitmarkIssue); ok {
				allAssets[tx.AssetId] = struct{}{}
			}
		}
	}

	// pending

	for _, item := range globalData.pendingFreeIssues {
		for _, tx := range item.txs {
			if tx, ok := tx.transaction.(*transactionrecord.BitmarkIssue); ok {
				allAssets[tx.AssetId] = struct{}{}
			}
		}
	}
	for _, item := range globalData.pendingPaidIssues {
		for _, tx := range item.txs {
			if tx, ok := tx.transaction.(*transactionrecord.BitmarkIssue); ok {
				allAssets[tx.AssetId] = struct{}{}
			}
		}
	}

	// save pending assets
backup_loop:
	for assetId := range allAssets {
		packedAsset := asset.Get(assetId)
		if nil == packedAsset {
			globalData.log.Errorf("asset [%s]: not in pending buffer", assetId)
			continue backup_loop // skip the corresponding issue since asset is corrupt
		}
		err := writeRecord(f, taggedTransaction, packedAsset)
		if nil != err {
			return err
		}
	}
	return nil
}

// pack up a proof record
func packProof(payId pay.PayId, nonce PayNonce) []byte {

	lp := len(payId)
	ln := len(nonce)
	packed := make([]byte, lp+ln)
	copy(packed[:], payId[:])
	copy(packed[lp:], nonce[:])

	return packed
}

// write a tagged block record
func writeBlock(f *os.File, tag tagType, txs []*transactionData) error {
	buffer := make([]byte, 0, 65535)
	for _, tx := range txs {
		buffer = append(buffer, tx.packed...)
	}
	return writeRecord(f, tag, buffer)
}

// write a tagged record
func writeRecord(f *os.File, tag tagType, packed []byte) error {

	if len(packed) > 65535 {
		globalData.log.Criticalf("write record packed length: %d > 65535", len(packed))
		logger.Panicf("write record packed length: %d > 65535", len(packed))
	}

	_, err := f.Write([]byte{byte(tag)})
	if nil != err {
		return err
	}

	count := make([]byte, 2)
	binary.BigEndian.PutUint16(count, uint16(len(packed)))
	_, err = f.Write(count)
	if nil != err {
		return err
	}
	_, err = f.Write(packed)
	return err
}

func readRecord(f *os.File) (tagType, transactionrecord.Packed, error) {

	tag := make([]byte, 1)
	n, err := f.Read(tag)
	if nil != err {
		return taggedEOF, []byte{}, err
	}
	if 1 != n {
		return taggedEOF, []byte{}, fmt.Errorf("read record name: read: %d, expected: %d", n, 1)
	}

	countBuffer := make([]byte, 2)
	n, err = f.Read(countBuffer)
	if nil != err {
		return taggedEOF, []byte{}, err
	}
	if 2 != n {
		return taggedEOF, []byte{}, fmt.Errorf("read record key count: read: %d, expected: %d", n, 2)
	}

	count := int(binary.BigEndian.Uint16(countBuffer))

	if count > 0 {
		buffer := make([]byte, count)
		n, err := f.Read(buffer)
		if nil != err {
			return taggedEOF, []byte{}, err
		}
		if count != n {
			return taggedEOF, []byte{}, fmt.Errorf("read record read: %d, expected: %d", n, count)
		}
		return tagType(tag[0]), buffer, nil
	}
	return tagType(tag[0]), []byte{}, nil
}
