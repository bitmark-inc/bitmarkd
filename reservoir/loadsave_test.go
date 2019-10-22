package reservoir_test

import (
	"crypto/ed25519"
	"encoding/binary"
	"fmt"
	"os"
	"testing"

	"github.com/bitmark-inc/bitmarkd/merkle"

	"github.com/bitmark-inc/bitmarkd/fault"

	"github.com/bitmark-inc/bitmarkd/asset"

	"github.com/bitmark-inc/bitmarkd/currency"

	"github.com/bitmark-inc/bitmarkd/reservoir/mocks"

	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/logger"

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
	seed, _ := account.NewBase58EncodedSeedV2(true)
	p, _ := account.PrivateKeyFromBase58Seed(seed)
	privateKey = p.PrivateKeyBytes()
	publicKey = p.Account().PublicKeyBytes()

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
		Fingerprint: "0123456789abcdefg",
		Metadata:    "owner\x00me",
		Name:        "asset name",
		Registrant:  &owner,
	}
	packed, err := assetData.Pack(&owner)
	if fault.InvalidSignature != err {
		fmt.Printf("pack asset data err: %s\n", err)
	}
	assetData.Signature = ed25519.Sign(privateKey, packed)
	assetID = assetData.AssetId()
	assetIssuance := transactionrecord.BitmarkIssue{
		AssetId:   assetID,
		Owner:     &owner,
		Nonce:     1,
		Signature: nil,
	}
	packed, err = assetIssuance.Pack(&owner)
	if fault.InvalidSignature != err {
		fmt.Printf("issuance pack err: %s\n", err)
	}
	signature := ed25519.Sign(privateKey, packed)
	assetIssuance.Signature = signature[:]
	p2, err := assetIssuance.Pack(&owner)
	if nil != err {
		fmt.Printf("second pack err: %s\n", err)
	}
	assetTxID = p2.MakeLink()
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
