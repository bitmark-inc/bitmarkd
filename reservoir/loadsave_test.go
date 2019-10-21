package reservoir_test

import (
	"crypto/ed25519"
	"encoding/binary"
	"fmt"
	"os"
	"testing"

	"github.com/bitmark-inc/bitmarkd/asset"

	"github.com/bitmark-inc/bitmarkd/currency"

	"github.com/bitmark-inc/bitmarkd/reservoir/mocks"

	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/logger"

	"github.com/bitmark-inc/bitmarkd/merkle"

	"github.com/bitmark-inc/bitmarkd/reservoir"

	"github.com/bitmark-inc/bitmarkd/account"

	"github.com/bitmark-inc/bitmarkd/transactionrecord"
)

const (
	dataFile   = "test.cache"
	loggerFile = "test.log"
)

const (
	taggedBOF byte = iota
	taggedEOF
	taggedTransaction
	taggedProof
	assetIDString = "0123456789012345678901234567890123456789012345678901234567890123"
	privateString = "6396dd14d2381e00682feb2a1b3171584361d70495abd33a43d6151a442d1bed"
)

var (
	bofData     = []byte("bitmark-cache v1.0")
	eofData     = []byte("EOF")
	assetID     transactionrecord.AssetIdentifier
	assetData   transactionrecord.AssetData
	owner       account.Account
	publicKey   []byte
	privateKey  []byte
	assetTxID   merkle.Digest
	currencyMap currency.Map
)

func removeLogger() {
	os.RemoveAll(loggerFile)
}

func setupLogger() {
	_ = logger.Initialise(logger.Configuration{
		Directory: ".",
		File:      loggerFile,
		Size:      50000,
		Count:     10,
	})
}

func init() {
	_, _ = fmt.Sscan(assetIDString, &assetID)

	publicKey = []byte{
		0x9f, 0xc4, 0x86, 0xa2, 0x53, 0x4f, 0x17, 0xe3,
		0x67, 0x07, 0xfa, 0x4b, 0x95, 0x3e, 0x3b, 0x34,
		0x00, 0xe2, 0x72, 0x9f, 0x65, 0x61, 0x16, 0xdd,
		0x7b, 0x01, 0x8d, 0xf3, 0x46, 0x98, 0xbd, 0xc2,
	}

	privateKey = []byte{
		0xf3, 0xf7, 0xa1, 0xfc, 0x33, 0x10, 0x71, 0xc2,
		0xb1, 0xcb, 0xbe, 0x4f, 0x3a, 0xee, 0x23, 0x5a,
		0xae, 0xcc, 0xd8, 0x5d, 0x2a, 0x80, 0x4c, 0x44,
		0xb5, 0xc6, 0x03, 0xb4, 0xca, 0x4d, 0x9e, 0xc0,
		0x9f, 0xc4, 0x86, 0xa2, 0x53, 0x4f, 0x17, 0xe3,
		0x67, 0x07, 0xfa, 0x4b, 0x95, 0x3e, 0x3b, 0x34,
		0x00, 0xe2, 0x72, 0x9f, 0x65, 0x61, 0x16, 0xdd,
		0x7b, 0x01, 0x8d, 0xf3, 0x46, 0x98, 0xbd, 0xc2,
	}

	assetTxID = merkle.Digest{
		0x54, 0x21, 0x45, 0x4d, 0x44, 0x9c, 0x63, 0x13,
		0x59, 0x48, 0x67, 0x19, 0x21, 0xdb, 0x9a, 0x7b,
		0xe2, 0x60, 0xb6, 0xab, 0x1f, 0x5c, 0x1c, 0x01,
		0x4f, 0x25, 0x14, 0x04, 0x08, 0x99, 0x85, 0x1c,
	}

	owner = account.Account{
		AccountInterface: &account.ED25519Account{
			Test:      true,
			PublicKey: publicKey,
		},
	}

	currencyMap = make(currency.Map)
	currencyMap[currency.Bitcoin] = "2N7uK4otZGYDUDNEQ3Yr6hPPrs49BHQA32L"
	currencyMap[currency.Litecoin] = "mwLH3WTj4zxMSM3Tzq3w9rfgJicawtKp1R"

	assetData = transactionrecord.AssetData{
		Name:        "asset name",
		Fingerprint: "0123456789abcdefg",
		Metadata:    "owner\x00me",
		Registrant:  &owner,
	}
	packed, _ := assetData.Pack(&owner)
	assetData.Signature = ed25519.Sign(privateKey, packed)
}

func initPackages() {
	_ = asset.Initialise()
}

func setupBackupFile() {
	f, _ := os.OpenFile(dataFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	defer f.Close()

	// begin of file
	var count, packed []byte
	_, _ = f.Write([]byte{byte(taggedBOF)})
	count = make([]byte, 2)
	packed = []byte(bofData)
	binary.BigEndian.PutUint16(count, uint16(len(packed)))
	_, _ = f.Write(count)
	_, _ = f.Write(packed)

	// asset issuance
	issue := transactionrecord.BitmarkIssue{
		AssetId: assetID,
		Owner:   &owner,
		Nonce:   1,
	}
	msg, _ := issue.Pack(&owner)
	issue.Signature = ed25519.Sign(privateKey, msg)

	_, _ = f.Write([]byte{byte(taggedTransaction)})
	packed, _ = issue.Pack(&owner)
	binary.BigEndian.PutUint16(count, uint16(len(packed)))
	_, _ = f.Write(count)
	_, _ = f.Write(packed)

	// asset data
	_, _ = f.Write([]byte{byte(taggedTransaction)})
	packed, _ = assetData.Pack(&owner)
	binary.BigEndian.PutUint16(count, uint16(len(packed)))
	_, _ = f.Write(count)
	_, _ = f.Write(packed)

	// end of file
	_, _ = f.Write([]byte{byte(taggedEOF)})
	packed = []byte(eofData)
	binary.BigEndian.PutUint16(count, uint16(len(packed)))
	_, _ = f.Write(count)
	_, _ = f.Write(packed)
}

func teardownDataFile() {
	_ = os.Remove(dataFile)
}

func setupMocks(t *testing.T) (*gomock.Controller, *mocks.MockHandle, *mocks.MockHandle) {
	ctl := gomock.NewController(t)

	mockAsset := mocks.NewMockHandle(ctl)
	mockBlockOwnerPayment := mocks.NewMockHandle(ctl)
	return ctl, mockAsset, mockBlockOwnerPayment
}

func TestLoadFromFileWhenAssetIssuance(t *testing.T) {
	setup(t, "testing")
	defer teardown()

	setupBackupFile()
	defer teardownDataFile()

	initPackages()
	defer asset.Finalise()

	ctl, mockAsset, mockBlockOwnerPayment := setupMocks(t)
	defer ctl.Finish()

	mockAsset.EXPECT().Has(gomock.Any()).Return(true).AnyTimes()
	mockAsset.EXPECT().GetNB(gomock.Any()).Return(uint64(2), []byte("exist")).Times(1)

	data, _ := currencyMap.Pack(true)
	mockBlockOwnerPayment.EXPECT().Get(gomock.Any()).Return(data).Times(1)

	_ = reservoir.Initialise(dataFile)
	_ = reservoir.LoadFromFile(mockAsset, mockBlockOwnerPayment)

	state := reservoir.TransactionStatus(assetTxID)
	assert.Equal(t, reservoir.StatePending, state, "wrong asset state")
}

func TestLoadFromFileWhenAssetData(t *testing.T) {
	setup(t, "testing")
	defer teardown()

	setupBackupFile()
	defer teardownDataFile()

	initPackages()
	defer asset.Finalise()

	ctl, mockAsset, mockBlockOwnerPayment := setupMocks(t)
	defer ctl.Finish()

	mockAsset.EXPECT().Has(gomock.Any()).Return(false).AnyTimes()

	_ = reservoir.Initialise(dataFile)
	_ = reservoir.LoadFromFile(mockAsset, mockBlockOwnerPayment)

	fmt.Printf("test id: %v\n", assetData.AssetId())
	result := asset.Exists(assetData.AssetId(), mockAsset)
	assert.Equal(t, true, result, "wrong asset cache")
}
