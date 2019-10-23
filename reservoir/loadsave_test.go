package reservoir_test

import (
	"crypto/ed25519"
	"encoding/binary"
	"fmt"
	"os"
	"testing"

	"github.com/bitmark-inc/bitmarkd/chain"
	"github.com/bitmark-inc/bitmarkd/ownership"

	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/storage"

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

type handles struct {
	asset             *mocks.MockHandle
	blockOwnerPayment *mocks.MockHandle
	transaction       *mocks.MockHandle
	ownerTx           *mocks.MockHandle
	ownerData         *mocks.MockHandle
}

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
	bofData          = []byte("bitmark-cache v1.0")
	eofData          = []byte("EOF")
	assetID          transactionrecord.AssetIdentifier
	assetData        transactionrecord.AssetData
	owner            account.Account
	publicKey        []byte
	privateKey       []byte
	assetTxID        merkle.Digest
	assetIssuance    transactionrecord.BitmarkIssue
	currencyMap      currency.Map
	txUnratifiedData transactionrecord.BitmarkTransferUnratified
	txUnratifiedID   merkle.Digest
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
	assetIssuance = transactionrecord.BitmarkIssue{
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
		fmt.Printf("second asset pack err: %s\n", err)
	}
	assetTxID = p2.MakeLink()

	txUnratifiedData = transactionrecord.BitmarkTransferUnratified{
		Link:      assetTxID,
		Escrow:    nil,
		Owner:     &owner,
		Signature: nil,
	}
	packed, err = txUnratifiedData.Pack(&owner)
	if fault.InvalidSignature != err {
		fmt.Printf("tx unratified pack err: %s\n", err)
	}
	signature = ed25519.Sign(privateKey, packed)
	txUnratifiedData.Signature = signature[:]
	p2, err = txUnratifiedData.Pack(&owner)
	if nil != err {
		fmt.Printf("second tx pack err: %s\n", err)
	}
	txUnratifiedID = p2.MakeLink()
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

	// transfer unratifierd
	_, _ = f.Write([]byte{byte(taggedTransaction)})
	packed, _ = txUnratifiedData.Pack(&owner)
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

func setupMocks(t *testing.T) ([]*gomock.Controller, handles) {
	var ctls []*gomock.Controller
	ctl1 := gomock.NewController(t)
	ctl2 := gomock.NewController(t)
	ctl3 := gomock.NewController(t)
	ctl4 := gomock.NewController(t)
	ctl5 := gomock.NewController(t)

	ctls = append(ctls, ctl1, ctl2, ctl3, ctl4, ctl5)

	return ctls, handles{
		asset:             mocks.NewMockHandle(ctl1),
		blockOwnerPayment: mocks.NewMockHandle(ctl2),
		transaction:       mocks.NewMockHandle(ctl3),
		ownerTx:           mocks.NewMockHandle(ctl4),
		ownerData:         mocks.NewMockHandle(ctl5),
	}
}

func TestLoadFromFileWhenAssetIssuance(t *testing.T) {
	setup(t, chain.Testing)
	defer teardown()

	setupBackupFile()
	defer teardownDataFile()

	initPackages()
	defer asset.Finalise()

	ctls, mockHandles := setupMocks(t)
	defer func(ctls []*gomock.Controller) {
		for _, c := range ctls {
			c.Finish()
		}
	}(ctls)

	mockHandles.asset.EXPECT().Has(gomock.Any()).Return(true).AnyTimes()
	mockHandles.asset.EXPECT().GetNB(gomock.Any()).Return(uint64(2), []byte("exist")).Times(1)

	data, _ := currencyMap.Pack(true)
	mockHandles.blockOwnerPayment.EXPECT().Get(gomock.Any()).Return(data).Times(1)

	mockHandles.transaction.EXPECT().GetNB(gomock.Any()).Return(uint64(2), []byte{}).AnyTimes()

	_ = reservoir.Initialise(dataFile)
	_ = reservoir.LoadFromFile(mockHandles.asset, mockHandles.blockOwnerPayment, mockHandles.transaction, mockHandles.ownerTx, storage.Pool.OwnerData)

	state := reservoir.TransactionStatus(assetTxID)
	assert.Equal(t, reservoir.StatePending, state, "wrong asset state")
}

func TestLoadFromFileWhenAssetData(t *testing.T) {
	setup(t, chain.Testing)
	defer teardown()

	setupBackupFile()
	defer teardownDataFile()

	initPackages()
	defer asset.Finalise()

	ctls, mockHandles := setupMocks(t)
	defer func(ctls []*gomock.Controller) {
		for _, c := range ctls {
			c.Finish()
		}
	}(ctls)

	mockHandles.asset.EXPECT().Has(gomock.Any()).Return(false).AnyTimes()

	mockHandles.transaction.EXPECT().GetNB(gomock.Any()).Return(uint64(2), []byte{}).AnyTimes()

	_ = reservoir.Initialise(dataFile)
	_ = reservoir.LoadFromFile(mockHandles.asset, mockHandles.blockOwnerPayment, mockHandles.transaction, mockHandles.ownerTx, storage.Pool.OwnerData)

	result := asset.Exists(assetData.AssetId(), mockHandles.asset)
	assert.Equal(t, true, result, "wrong asset cache")
}

func TestLoadFromFileWhenTransferUnratified(t *testing.T) {
	setup(t, chain.Testing)
	defer teardown()

	setupBackupFile()
	defer teardownDataFile()

	initPackages()
	defer asset.Finalise()

	ctls, mockHandles := setupMocks(t)
	defer func(ctls []*gomock.Controller) {
		for _, c := range ctls {
			c.Finish()
		}
	}(ctls)

	mockHandles.asset.EXPECT().Has(gomock.Any()).Return(true).Times(1)
	mockHandles.asset.EXPECT().GetNB(gomock.Any()).Return(uint64(2), []byte("exist")).AnyTimes()

	data, _ := currencyMap.Pack(true)
	mockHandles.blockOwnerPayment.EXPECT().Get(gomock.Any()).Return(data).AnyTimes()

	packed, err := assetIssuance.Pack(&owner)
	if nil != err {
		fmt.Printf("asset pack err: %s\n", err)
	}

	mockHandles.transaction.EXPECT().GetNB(gomock.Any()).Return(uint64(2), packed).AnyTimes()
	mockHandles.transaction.EXPECT().Has(gomock.Any()).Return(false).AnyTimes()

	mockHandles.ownerTx.EXPECT().Get(gomock.Any()).Return([]byte("1")).Times(1)

	packedOwnerData := ownership.PackedOwnerData{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x30,
		0x39, 0xa7, 0x4a, 0x90, 0xc2, 0xff, 0x76, 0x34,
		0x7a, 0x9d, 0x34, 0x19, 0xe9, 0x20, 0x2f, 0x02,
		0xd8, 0xff, 0x5d, 0xdd, 0xa2, 0x7c, 0xc1, 0x7b,
		0xa1, 0x71, 0xbc, 0x7c, 0x68, 0xbc, 0xc9, 0xce,
		0x49, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04,
		0xd2, 0x59, 0xd0, 0x61, 0x55, 0xd2, 0x5d, 0xff,
		0xdb, 0x98, 0x27, 0x29, 0xde, 0x8d, 0xce, 0x9d,
		0x78, 0x55, 0xca, 0x09, 0x4d, 0x8b, 0xab, 0x81,
		0x24, 0xb3, 0x47, 0xc4, 0x06, 0x68, 0x47, 0x70,
		0x56, 0xb3, 0xc2, 0x7c, 0xcb, 0x7d, 0x71, 0xb5,
		0x40, 0x43, 0xd2, 0x07, 0xcc, 0xd1, 0x87, 0x64,
		0x2b, 0xf9, 0xc8, 0x46, 0x6f, 0x9a, 0x8d, 0x0d,
		0xbe, 0xfb, 0x4c, 0x41, 0x63, 0x3a, 0x7e, 0x39,
		0xef,
	}
	mockHandles.ownerData.EXPECT().Get(gomock.Any()).Return(packedOwnerData).AnyTimes()

	_ = reservoir.Initialise(dataFile)
	_ = reservoir.LoadFromFile(mockHandles.asset, mockHandles.blockOwnerPayment, mockHandles.transaction, mockHandles.ownerTx, mockHandles.ownerData)

	result := reservoir.TransactionStatus(txUnratifiedID)
	assert.Equal(t, reservoir.StatePending, result, "wrong transfer state")
}
