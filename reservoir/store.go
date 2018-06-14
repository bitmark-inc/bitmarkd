// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package reservoir

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/bitmark-inc/bitmarkd/asset"
	"github.com/bitmark-inc/bitmarkd/cache"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
)

var (
	ErrBackupBeforeInit = fmt.Errorf("can not backup data before initialisation")
	ErrLoadMoreThanOnce = fmt.Errorf("can not recover data more than once")
)

const (
	anAsset   = "A"
	anIssue   = "I"
	aTransfer = "T"
	aProof    = "P"
)

// ReservoirStore is the struct for backup and recover unconfirmed issues
// and transfer
type ReservoirStore struct {
	initialised bool
	filename    string
}

// NewReservoirStore returns a ReservoirStore with a given backup file
func NewReservoirStore(filename string) *ReservoirStore {
	return &ReservoirStore{
		filename: filename,
	}
}

// Backup is to save current pending and verified items to disk
func (rs ReservoirStore) Backup() error {
	if !rs.initialised {
		return ErrBackupBeforeInit
	}

	f, err := os.OpenFile(rs.filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if nil != err {
		return err
	}
	defer f.Close()

	err = backupAssets(f)
	if nil != err {
		return err
	}

	return backupTransactions(f)
}

// Restore is to recover pending and verified items from disk
func (rs *ReservoirStore) Restore() error {
	defer func() {
		rs.initialised = true
	}()

	if rs.initialised {
		return ErrLoadMoreThanOnce
	}

	f, err := os.OpenFile(rs.filename, os.O_RDONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

restore_loop:
	for {
		name, packed, err := readRecord(f)
		if io.EOF == err {
			break restore_loop
		} else if nil != err {
			return err
		}
		switch name {
		case anAsset:
			transaction, _, err := packed.Unpack(mode.IsTesting())
			if nil != err {
				globalData.log.Errorf("unable to unpack asset: %s", err)
				continue restore_loop
			}

			if assetData, ok := transaction.(*transactionrecord.AssetData); ok {
				_, _, err := asset.Cache(assetData)
				if nil != err {
					globalData.log.Errorf("fail to cache asset: %s", err)
				}
			} else {
				globalData.log.Errorf("invalid type casting of assetData: %+v", transaction)
			}

		case anIssue:
			packedIssues := packed
			issues := make([]*transactionrecord.BitmarkIssue, 0, 100)
			for len(packedIssues) > 0 {
				transaction, n, err := packedIssues.Unpack(mode.IsTesting())
				if nil != err {
					globalData.log.Errorf("unable to unpack issue: %s", err)
					continue restore_loop
				}

				if issue, ok := transaction.(*transactionrecord.BitmarkIssue); ok {
					issues = append(issues, issue)
				} else {
					globalData.log.Errorf("invalid type casting of bitmarkIssue: %+v", transaction)
				}
				packedIssues = packedIssues[n:]
			}

			_, _, err := StoreIssues(issues)
			if nil != err {
				globalData.log.Errorf("fail to store issue: %s", err)
			}

		case aTransfer:
			transaction, _, err := packed.Unpack(mode.IsTesting())
			if nil != err {
				globalData.log.Errorf("unable to unpack transfer: %s", err)
				continue restore_loop
			}

			transfer, ok := transaction.(transactionrecord.BitmarkTransfer)
			if ok {
				_, _, err := StoreTransfer(transfer)
				if nil != err {
					globalData.log.Errorf("fail to store transfer: %s", err)
				}
			} else {
				globalData.log.Errorf("invalid type casting of bitmarkTransfer: %+v", transaction)
			}

		case aProof:
			var payId pay.PayId
			pn := len(payId)
			if len(packed) <= pn {
				globalData.log.Errorf("unable to unpack proof: record too short: %d  expected > %d", len(packed), pn)
				continue restore_loop
			}
			copy(payId[:], packed[:pn])
			nonce := packed[pn:]
			TryProof(payId, nonce)
		}
	}

	return nil
}

func backupAssets(f *os.File) error {
	allAssets := make(map[transactionrecord.AssetIdentifier]struct{})

	// extract all asset ids from unverified items
	for _, val := range cache.Pool.UnverifiedTxEntries.Items() {
		v := val.(*unverifiedItem)
		if v.links == nil {
			for assetId := range v.itemData.assetIds {
				allAssets[assetId] = struct{}{}
			}
		}
	}

	// backup all verified items
	for _, val := range cache.Pool.VerifiedTx.Items() {
		v := val.(*verifiedItem)
		if v.links == nil && 0 == v.index { // only need to check assets on first record of block
			for assetId := range v.itemData.assetIds {
				allAssets[assetId] = struct{}{}
			}
		}
	}

backup_loop:
	for assetId := range allAssets {
		packedAsset, err := fetchAsset(assetId)
		if nil != err {
			globalData.log.Errorf("asset [%s]: error: %s", assetId, err)
			continue backup_loop // skip the corresponding issue since asset is corrupt
		}
		err = writeRecord(f, anAsset, packedAsset)
		if nil != err {
			return err
		}
	}
	return nil
}

// backup is to backup all unverified / verified items
func backupTransactions(f *os.File) (err error) {

	// backup all unverified items
	for _, val := range cache.Pool.UnverifiedTxEntries.Items() {
		v := val.(*unverifiedItem)
		if nil != v.links {
			err = writeRecord(f, aTransfer, v.transactions[0])
		} else {
			packedIssues := bytes.Join(v.transactions, []byte{})
			err = writeRecord(f, anIssue, packedIssues)
		}
		if nil != err {
			return err
		}
	}

	// backup all verified items
	for _, val := range cache.Pool.VerifiedTx.Items() {
		v := val.(*verifiedItem)
		if nil != v.links {
			err = writeRecord(f, aTransfer, v.transaction)
		} else if 0 == v.index {
			packedIssues := bytes.Join(v.itemData.transactions, []byte{})
			err = writeRecord(f, anIssue, packedIssues)
			if nil != err {
				return err
			}
			if nil != v.itemData.nonce {
				payId := pay.NewPayId(v.itemData.transactions)
				proof := append(payId[:], v.itemData.nonce...)
				err = writeRecord(f, aProof, proof)
			}
		}
		if nil != err {
			return err
		}
	}
	return nil
}

func writeRecord(f *os.File, name string, packed []byte) error {
	if 1 != len(name) {
		globalData.log.Criticalf("write record name error: %q", name)
		logger.Panicf("write record name error: %q", name)
	}

	if len(packed) > 65535 {
		globalData.log.Criticalf("write record packed length: %d > 65535", len(packed))
		logger.Panicf("write record packed length: %d > 65535", len(packed))
	}

	_, err := f.Write([]byte(name))
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

func readRecord(f *os.File) (string, transactionrecord.Packed, error) {

	name := make([]byte, 1)
	n, err := f.Read(name)
	if nil != err {
		return "", []byte{}, err
	}
	if 1 != n {
		return "", []byte{}, fmt.Errorf("read record name: read: %d, expected: %d", n, 1)
	}

	// data

	packed, err := readCounted(f)
	if nil != err {
		return "", []byte{}, err
	}

	return string(name), packed, nil
}

func readCounted(f *os.File) ([]byte, error) {

	buffer := []byte{}

	countBuffer := make([]byte, 2)
	n, err := f.Read(countBuffer)
	if nil != err {
		return buffer, err
	}
	if 2 != n {
		return buffer, fmt.Errorf("read record key count: read: %d, expected: %d", n, 2)
	}

	count := int(binary.BigEndian.Uint16(countBuffer))

	if count > 0 {
		buffer = make([]byte, count)
		n, err := f.Read(buffer)
		if nil != err {
			return buffer, err
		}
		if count != n {
			return buffer, fmt.Errorf("read record read: %d, expected: %d", n, count)
		}
	}
	return buffer, nil
}
