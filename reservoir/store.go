package reservoir

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/bitmark-inc/bitmarkd/asset"
	"github.com/bitmark-inc/bitmarkd/cache"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"os"
)

var (
	ErrBackupBeforeInit = fmt.Errorf("can not backup data before initialisation")
	ErrLoadMoreThanOnce = fmt.Errorf("can not recover data more than once")
)

// ReservoirStore is the struct for backup and recover unconfirmed issues
// and transfer
type ReservoirStore struct {
	init         bool
	filename     string
	Assets       map[transactionrecord.AssetIndex]transactionrecord.Packed
	Issues       []transactionrecord.Packed
	ProofFilters map[string]ProofFilter
	Transfers    []transactionrecord.Packed
}

// NewReservoirStore returns a ReservoirStore with a given backup file
func NewReservoirStore(filename string) *ReservoirStore {
	return &ReservoirStore{
		filename: filename,
	}
}

// Backup is to save current pending and verified items to disk
func (rs ReservoirStore) Backup() error {
	if !rs.init {
		return ErrBackupBeforeInit
	}

	rs.backup()

	f, err := os.OpenFile(rs.filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	return encoder.Encode(&rs)
}

// Restore is to recover pending and verified items from disk
func (rs *ReservoirStore) Restore() error {
	defer func() { rs.init = true }()
	if rs.init {
		return ErrLoadMoreThanOnce
	}
	f, err := os.OpenFile(rs.filename, os.O_RDONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	d := json.NewDecoder(f)
	err = d.Decode(&rs)
	if err != nil {
		return err
	}

	rs.internalRecover()
	return nil
}

// recover from the ReservoirStore. It put proof filters back and then
// re-broadcasts assets, issues and transfers.
func (rs *ReservoirStore) internalRecover() {
	globalData.log.Debugf("start recovering from data: %+v\n", rs)

	// Put back proof filters
	for payId, proofFilter := range rs.ProofFilters {
		cache.Pool.ProofFilters.Put(payId, proofFilter)
	}

ASSET_RECOVERY:
	for _, packedAsset := range rs.Assets {
		transaction, _, err := packedAsset.Unpack(mode.IsTesting())
		if nil != err {
			globalData.log.Errorf("unable to unpack asset: %s", err.Error())
			continue ASSET_RECOVERY
		}

		if assetData, ok := transaction.(*transactionrecord.AssetData); ok {
			_, _, err := asset.Cache(assetData)
			if nil != err {
				globalData.log.Errorf("fail to cache asset: %s", err.Error())
			}
		} else {
			globalData.log.Errorf("invalid type casting of assetData: %+v", transaction)
		}
	}

ISSUE_RECOVERY:
	for _, packedIssues := range rs.Issues {
		// Check whether an issue is in the filter or not.
		var verified bool
	CHECK_VERIFIED:
		for _, filter := range rs.ProofFilters {
			if filter.Has(packedIssues) {
				verified = true
				break CHECK_VERIFIED
			}
		}

		issues := make([]*transactionrecord.BitmarkIssue, 0, 1)
		for len(packedIssues) != 0 {
			transaction, n, err := packedIssues.Unpack(mode.IsTesting())
			if nil != err {
				globalData.log.Errorf("unable to unpack issue: %s", err.Error())
				continue ISSUE_RECOVERY
			}

			if issue, ok := transaction.(*transactionrecord.BitmarkIssue); ok {
				issues = append(issues, issue)
			} else {
				globalData.log.Errorf("invalid type casting of bitmarkIssue: %+v", transaction)
			}
			packedIssues = packedIssues[n:]
		}

		_, _, err := StoreIssues(issues, verified)
		if nil != err {
			globalData.log.Errorf("fail to store issue: %s", err.Error())
		}
	}

TRANSFER_RECOVERY:
	for _, packedTransfer := range rs.Transfers {
		transaction, _, err := packedTransfer.Unpack(mode.IsTesting())
		if nil != err {
			globalData.log.Errorf("unable to unpack transfer: %s", err.Error())
			continue TRANSFER_RECOVERY
		}
		if transaction.IsTransfer() {
			_, _, err := StoreTransfer(transaction.(transactionrecord.BitmarkTransfer))
			if nil != err {
				globalData.log.Errorf("fail to store transfer: %s", err.Error())
			}
		} else {
			globalData.log.Errorf("invalid type casting of bitmarkTransfer: %+v", transaction)
		}
	}
}

// backup is to backup all unverified / verified items
func (rs *ReservoirStore) backup() {
	packedAssets := map[transactionrecord.AssetIndex]transactionrecord.Packed{}
	packedIssues := []transactionrecord.Packed{}
	packedTransfer := []transactionrecord.Packed{}

	// backup all unverified items
	for _, val := range cache.Pool.UnverifiedTxEntries.Items() {
		v := val.(*unverifiedItem)
		if v.links == nil {
			// all assets for unverified issues
			for _, assetId := range v.itemData.assetIds {
				if _, ok := packedAssets[assetId]; !ok {
					packedAsset, err := fetchAsset(assetId)
					if err != nil {
						globalData.log.Errorf("asset id[%s]: error: %s", assetId, err.Error())
					} else {
						packedAssets[assetId] = packedAsset
					}
				}
			}
			packedIssues = append(packedIssues, bytes.Join(v.transactions, []byte{}))
		} else {
			// currently will inlcude one and only transfer
			packedTransfer = append(packedTransfer, v.transactions[0])
		}
	}

	// backup all verified items
backup_loop:
	for _, val := range cache.Pool.VerifiedTx.Items() {
		v := val.(*verifiedItem)
		if v.links == nil {
			// verified issues
			assetId := v.itemData.assetIds[v.index]
			if _, ok := packedAssets[assetId]; !ok {
				packedAsset, err := fetchAsset(assetId)
				if err != nil {
					globalData.log.Errorf("asset [%s]: error: %s", assetId, err.Error())
					continue backup_loop // skip the corresponding issue since asset is corrupt
				} else {
					packedAssets[assetId] = packedAsset
				}
			}
			packedIssues = append(packedIssues, v.transaction)
		} else {
			packedTransfer = append(packedTransfer, v.transaction)
		}
	}

	packedProofFilters := map[string]ProofFilter{}
	for key, val := range cache.Pool.ProofFilters.Items() {
		packedProofFilters[key] = val.(ProofFilter)
	}

	rs.Assets = packedAssets
	rs.Issues = packedIssues
	rs.ProofFilters = packedProofFilters
	rs.Transfers = packedTransfer
}
