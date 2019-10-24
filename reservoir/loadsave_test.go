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
	share             *mocks.MockHandle
	shareQuantity     *mocks.MockHandle
}

const (
	dataFile      = "test.cache"
	loggerFile    = "test.log"
	beforeBlock   = 5
	shareQuantity = 100
)

const (
	taggedBOF byte = iota
	taggedEOF
	taggedTransaction
	taggedProof
)

var (
	bofData = []byte("bitmark-cache v1.0")
	eofData = []byte("EOF")

	// asset
	assetIssuance transactionrecord.BitmarkIssue
	assetID       transactionrecord.AssetIdentifier
	assetData     transactionrecord.AssetData
	assetTxID     merkle.Digest

	// owner
	owner           account.Account
	publicKey       []byte
	privateKey      []byte
	owner2          account.Account
	publicKey2      []byte
	privateKey2     []byte
	packedOwnerData ownership.PackedOwnerData

	// payment
	currencyMap currency.Map

	// transfer unratified
	txUnratifiedData transactionrecord.BitmarkTransferUnratified
	txUnratifiedID   merkle.Digest

	// share
	txShareData transactionrecord.BitmarkShare
	txShareID   merkle.Digest

	// grant
	grantData transactionrecord.ShareGrant
	grantID   merkle.Digest

	// swat
	swapData transactionrecord.ShareSwap
	swapID   merkle.Digest
)

func init() {
	// owner
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

	// owner2
	seed, _ = account.NewBase58EncodedSeedV2(true)
	p, _ = account.PrivateKeyFromBase58Seed(seed)
	privateKey2 = p.PrivateKeyBytes()
	publicKey2 = p.Account().PublicKeyBytes()
	owner2 = account.Account{
		AccountInterface: &account.ED25519Account{
			Test:      true,
			PublicKey: publicKey2,
		},
	}
	// payment
	currencyMap = make(currency.Map)
	currencyMap[currency.Bitcoin] = "2N7uK4otZGYDUDNEQ3Yr6hPPrs49BHQA32L"
	currencyMap[currency.Litecoin] = "mwLH3WTj4zxMSM3Tzq3w9rfgJicawtKp1R"

	// packed owner data
	// 	AssetOwnerData{
	//		transferBlockNumber: 12345,
	//		issueTxId:           49cec9bc687cbc71a17bc17ca2dd5dffd8022f20e919349d7a3476ffc2904aa7,
	//		issueBlockNumber:    1234,
	//		assetId:             59d06155d25dffdb982729de8dce9d7855ca094d8bab8124b347c40668477056b3c27ccb7d71b54043d207ccd187642bf9c8466f9a8d0dbefb4c41633a7e39ef,
	//	}
	packedOwnerData = ownership.PackedOwnerData{
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

	// asset
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

	// transfer unratified
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

	// create share
	txShareData = transactionrecord.BitmarkShare{
		Link:      txUnratifiedID,
		Quantity:  shareQuantity,
		Signature: nil,
	}
	packed, err = txShareData.Pack(&owner)
	if fault.InvalidSignature != err {
		fmt.Printf("share pack error: %s\n", err)
	}
	signature = ed25519.Sign(privateKey, packed)
	txShareData.Signature = signature[:]
	p2, err = txShareData.Pack(&owner)
	if nil != err {
		fmt.Printf("second share pack err: %s\n", err)
	}
	txShareID = p2.MakeLink()

	// grant
	grantData = transactionrecord.ShareGrant{
		ShareId:          txShareID,
		Quantity:         100,
		Owner:            &owner,
		Recipient:        &owner2,
		BeforeBlock:      beforeBlock,
		Signature:        nil,
		Countersignature: nil,
	}
	packed, err = grantData.Pack(&owner)
	if fault.InvalidSignature != err {
		fmt.Printf("grant pack eerror: %s\n", err)
	}
	signature = ed25519.Sign(privateKey, packed)
	grantData.Signature = signature[:]
	packed, err = grantData.Pack(&owner)
	if fault.InvalidSignature != err {
		fmt.Printf("second grant pack err: %s\n", err)
	}
	signature = ed25519.Sign(privateKey2, packed)
	grantData.Countersignature = signature[:]
	p2, err = grantData.Pack(&owner)
	if nil != err {
		fmt.Printf("second grant err: %s\n", err)
	}
	grantID = p2.MakeLink()

	// swap
	swapData = transactionrecord.ShareSwap{
		ShareIdOne:       txShareID,
		QuantityOne:      shareQuantity,
		OwnerOne:         &owner,
		ShareIdTwo:       txUnratifiedID,
		QuantityTwo:      shareQuantity,
		OwnerTwo:         &owner2,
		BeforeBlock:      beforeBlock,
		Signature:        nil,
		Countersignature: nil,
	}
	packed, err = swapData.Pack(&owner)
	if fault.InvalidSignature != err {
		fmt.Printf("swap pack err: %s\n", err)
	}
	signature = ed25519.Sign(privateKey, packed)
	swapData.Signature = signature[:]
	packed, err = swapData.Pack(&owner)
	if fault.InvalidSignature != err {
		fmt.Printf("swap pack 2 err: %s\n", err)
	}
	signature = ed25519.Sign(privateKey2, packed)
	swapData.Countersignature = signature
	packed, err = swapData.Pack(&owner)
	if nil != err {
		fmt.Printf("swap pack 3 err: %s\n", err)
	}
	swapID = packed.MakeLink()
}

func initPackages() {
	_ = asset.Initialise()
}

func writeBeginOfFile(f *os.File) {
	count := make([]byte, 2)
	_, _ = f.Write([]byte{taggedBOF})
	binary.BigEndian.PutUint16(count, uint16(len(bofData)))
	_, _ = f.Write(count)
	_, _ = f.Write(bofData)
}

func writeEndOfFile(f *os.File) {
	count := make([]byte, 2)
	_, _ = f.Write([]byte{taggedEOF})
	binary.BigEndian.PutUint16(count, uint16(len(eofData)))
	_, _ = f.Write(count)
	_, _ = f.Write(eofData)
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
	ctl6 := gomock.NewController(t)
	ctl7 := gomock.NewController(t)

	ctls = append(ctls, ctl1, ctl2, ctl3, ctl4, ctl5, ctl6)

	return ctls, handles{
		asset:             mocks.NewMockHandle(ctl1),
		blockOwnerPayment: mocks.NewMockHandle(ctl2),
		transaction:       mocks.NewMockHandle(ctl3),
		ownerTx:           mocks.NewMockHandle(ctl4),
		ownerData:         mocks.NewMockHandle(ctl5),
		share:             mocks.NewMockHandle(ctl6),
		shareQuantity:     mocks.NewMockHandle(ctl7),
	}
}

func finaliseMockController(ctls []*gomock.Controller) {
	for _, c := range ctls {
		c.Finish()
	}
}

func writeAssetIssuance(f *os.File) {
	count := make([]byte, 2)
	issue := transactionrecord.BitmarkIssue{
		AssetId: assetID,
		Owner:   &owner,
		Nonce:   1,
	}
	msg, _ := issue.Pack(&owner)
	issue.Signature = ed25519.Sign(privateKey, msg)
	_, _ = f.Write([]byte{taggedTransaction})
	packed, _ := issue.Pack(&owner)
	binary.BigEndian.PutUint16(count, uint16(len(packed)))
	_, _ = f.Write(count)
	_, _ = f.Write(packed)
}

func setupAssetIssuanceBackupFile() {
	f, _ := os.OpenFile(dataFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	defer f.Close()

	// begin of file
	writeBeginOfFile(f)

	// asset issuance
	writeAssetIssuance(f)

	// end of file
	writeEndOfFile(f)
}

func TestLoadFromFileWhenAssetIssuance(t *testing.T) {
	setup(t, chain.Testing)
	defer teardown()

	setupAssetIssuanceBackupFile()
	defer teardownDataFile()

	initPackages()
	defer asset.Finalise()

	ctls, mockHandles := setupMocks(t)
	defer finaliseMockController(ctls)

	mockHandles.asset.EXPECT().Has(gomock.Any()).Return(true).Times(1)
	mockHandles.asset.EXPECT().GetNB(gomock.Any()).Return(uint64(2), []byte("exist")).Times(1)

	data, _ := currencyMap.Pack(true)
	mockHandles.blockOwnerPayment.EXPECT().Get(gomock.Any()).Return(data).Times(1)

	_ = reservoir.Initialise(dataFile)
	_ = reservoir.LoadFromFile(mockHandles.asset, mockHandles.blockOwnerPayment, mockHandles.transaction, mockHandles.ownerTx, storage.Pool.OwnerData, mockHandles.shareQuantity, mockHandles.share)

	state := reservoir.TransactionStatus(assetTxID)
	assert.Equal(t, reservoir.StatePending, state, "wrong asset state")
}

func writeAssetData(f *os.File) {
	count := make([]byte, 2)
	_, _ = f.Write([]byte{taggedTransaction})
	packed, _ := assetData.Pack(&owner)
	binary.BigEndian.PutUint16(count, uint16(len(packed)))
	_, _ = f.Write(count)
	_, _ = f.Write(packed)
}

func setupAssetDataBackupFile() {
	f, _ := os.OpenFile(dataFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	defer f.Close()

	// begin of file
	writeBeginOfFile(f)

	// asset data
	writeAssetData(f)

	// end of file
	writeEndOfFile(f)
}

func TestLoadFromFileWhenAssetData(t *testing.T) {
	setup(t, chain.Testing)
	defer teardown()

	setupAssetDataBackupFile()
	defer teardownDataFile()

	initPackages()
	defer asset.Finalise()

	ctls, mockHandles := setupMocks(t)
	defer finaliseMockController(ctls)

	mockHandles.asset.EXPECT().Has(gomock.Any()).Return(false).Times(1)

	_ = reservoir.Initialise(dataFile)
	_ = reservoir.LoadFromFile(mockHandles.asset, mockHandles.blockOwnerPayment, mockHandles.transaction, mockHandles.ownerTx, storage.Pool.OwnerData, mockHandles.shareQuantity, mockHandles.share)

	result := asset.Exists(assetData.AssetId(), mockHandles.asset)
	assert.Equal(t, true, result, "wrong asset cache")
}

func writeTransferUnratified(f *os.File) {
	count := make([]byte, 2)
	_, _ = f.Write([]byte{taggedTransaction})
	packed, _ := txUnratifiedData.Pack(&owner)
	binary.BigEndian.PutUint16(count, uint16(len(packed)))
	_, _ = f.Write(count)
	_, _ = f.Write(packed)
}

func setupTransferUnratifiedBackupFile() {
	f, _ := os.OpenFile(dataFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	defer f.Close()

	// begin of file
	writeBeginOfFile(f)

	// transfer unratified
	writeTransferUnratified(f)

	// end of file
	writeEndOfFile(f)
}

func TestLoadFromFileWhenTransferUnratified(t *testing.T) {
	setup(t, chain.Testing)
	defer teardown()

	setupTransferUnratifiedBackupFile()
	defer teardownDataFile()

	initPackages()
	defer asset.Finalise()

	ctls, mockHandles := setupMocks(t)
	defer finaliseMockController(ctls)

	data, _ := currencyMap.Pack(true)
	mockHandles.blockOwnerPayment.EXPECT().Get(gomock.Any()).Return(data).AnyTimes()

	packed, err := assetIssuance.Pack(&owner)
	if nil != err {
		fmt.Printf("asset pack err: %s\n", err)
	}

	mockHandles.transaction.EXPECT().GetNB(gomock.Any()).Return(uint64(2), packed).Times(1)
	mockHandles.transaction.EXPECT().Has(gomock.Any()).Return(false).Times(1)

	mockHandles.ownerTx.EXPECT().Get(gomock.Any()).Return([]byte("1")).Times(1)

	mockHandles.ownerData.EXPECT().Get(gomock.Any()).Return(packedOwnerData).Times(1)

	_ = reservoir.Initialise(dataFile)
	_ = reservoir.LoadFromFile(mockHandles.asset, mockHandles.blockOwnerPayment, mockHandles.transaction, mockHandles.ownerTx, mockHandles.ownerData, mockHandles.shareQuantity, mockHandles.share)

	result := reservoir.TransactionStatus(txUnratifiedID)
	assert.Equal(t, reservoir.StatePending, result, "wrong transfer state")
}

func writeShareIssuance(f *os.File) {
	count := make([]byte, 2)
	_, _ = f.Write([]byte{taggedTransaction})
	packed, _ := txShareData.Pack(&owner)
	binary.BigEndian.PutUint16(count, uint16(len(packed)))
	_, _ = f.Write(count)
	_, _ = f.Write(packed)
}

func setupShareBackupFile() {
	f, _ := os.OpenFile(dataFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	defer f.Close()

	// begin of file
	writeBeginOfFile(f)

	// transfer unratified
	writeTransferUnratified(f)

	// transfer unratified
	writeShareIssuance(f)

	// end of file
	writeEndOfFile(f)
}

func TestLoadFromFileWhenShare(t *testing.T) {
	setup(t, chain.Testing)
	defer teardown()

	setupShareBackupFile()
	defer teardownDataFile()

	initPackages()
	defer asset.Finalise()

	ctls, mockHandles := setupMocks(t)
	defer finaliseMockController(ctls)

	data, _ := currencyMap.Pack(true)
	mockHandles.blockOwnerPayment.EXPECT().Get(gomock.Any()).Return(data).AnyTimes()

	packed, err := assetIssuance.Pack(&owner)
	if nil != err {
		fmt.Printf("asset pack err: %s\n", err)
	}

	mockHandles.transaction.EXPECT().GetNB(gomock.Any()).Return(uint64(2), packed).Times(2)
	mockHandles.transaction.EXPECT().Has(gomock.Any()).Return(false).Times(2)

	mockHandles.ownerTx.EXPECT().Get(gomock.Any()).Return([]byte("1")).Times(2)

	mockHandles.ownerData.EXPECT().Get(gomock.Any()).Return(packedOwnerData).Times(2)

	_ = reservoir.Initialise(dataFile)
	_ = reservoir.LoadFromFile(mockHandles.asset, mockHandles.blockOwnerPayment, mockHandles.transaction, mockHandles.ownerTx, mockHandles.ownerData, mockHandles.shareQuantity, mockHandles.share)

	result := reservoir.TransactionStatus(txShareID)
	assert.Equal(t, reservoir.StatePending, result, "wrong share state")
}

func writeGrant(f *os.File) {
	count := make([]byte, 2)
	_, _ = f.Write([]byte{taggedTransaction})
	packed, _ := grantData.Pack(&owner)
	binary.BigEndian.PutUint16(count, uint16(len(packed)))
	_, _ = f.Write(count)
	_, _ = f.Write(packed)
}

func setupGrantBackupFile() {
	f, _ := os.OpenFile(dataFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	defer f.Close()

	// begin of file
	writeBeginOfFile(f)

	// grant
	writeGrant(f)

	// end of file
	writeEndOfFile(f)
}

func TestLoadFromFileWhenGrant(t *testing.T) {
	setup(t, chain.Testing)
	defer teardown()

	setupGrantBackupFile()
	defer teardownDataFile()

	initPackages()
	defer asset.Finalise()

	ctls, mockHandles := setupMocks(t)
	defer finaliseMockController(ctls)

	data, _ := currencyMap.Pack(true)
	mockHandles.blockOwnerPayment.EXPECT().Get(gomock.Any()).Return(data).AnyTimes()

	mockHandles.ownerData.EXPECT().Get(gomock.Any()).Return(packedOwnerData).Times(1)
	mockHandles.shareQuantity.EXPECT().GetN(gomock.Any()).Return(uint64(shareQuantity), true).Times(1)
	mockHandles.share.EXPECT().GetNB(gomock.Any()).Return(uint64(shareQuantity), []byte{}).Times(1)

	_ = reservoir.Initialise(dataFile)
	err := reservoir.LoadFromFile(mockHandles.asset, mockHandles.blockOwnerPayment, mockHandles.transaction, mockHandles.ownerTx, mockHandles.ownerData, mockHandles.shareQuantity, mockHandles.share)
	if nil != err {
		fmt.Printf("load from file err: %s\n", err)

	}

	result := reservoir.TransactionStatus(grantID)
	assert.Equal(t, reservoir.StatePending, result, "wrong share state")
}

func writeSwap(f *os.File) {
	count := make([]byte, 2)
	_, _ = f.Write([]byte{taggedTransaction})
	packed, _ := swapData.Pack(&owner)
	binary.BigEndian.PutUint16(count, uint16(len(packed)))
	_, _ = f.Write(count)
	_, _ = f.Write(packed)
}

func setupSwapBackupFile() {
	f, _ := os.OpenFile(dataFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	defer f.Close()

	// begin of file
	writeBeginOfFile(f)

	// swap
	writeSwap(f)

	// end of file
	writeEndOfFile(f)
}

func TestLoadFromFileWhenSwap(t *testing.T) {
	setup(t, chain.Testing)
	defer teardown()

	setupSwapBackupFile()
	defer teardownDataFile()

	initPackages()
	defer asset.Finalise()

	ctls, mockHandles := setupMocks(t)
	defer finaliseMockController(ctls)

	data, _ := currencyMap.Pack(true)
	mockHandles.blockOwnerPayment.EXPECT().Get(gomock.Any()).Return(data).AnyTimes()

	mockHandles.shareQuantity.EXPECT().GetN(gomock.Any()).Return(uint64(shareQuantity), true).Times(2)
	mockHandles.share.EXPECT().GetNB(gomock.Any()).Return(uint64(shareQuantity), []byte("ok")).Times(1)
	mockHandles.ownerData.EXPECT().Get(gomock.Any()).Return(packedOwnerData).Times(1)

	_ = reservoir.Initialise(dataFile)
	err := reservoir.LoadFromFile(mockHandles.asset, mockHandles.blockOwnerPayment, mockHandles.transaction, mockHandles.ownerTx, mockHandles.ownerData, mockHandles.shareQuantity, mockHandles.share)
	if nil != err {
		fmt.Printf("load from file err: %s\n", err)
	}

	result := reservoir.TransactionStatus(swapID)
	assert.Equal(t, reservoir.StatePending, result, "wrong swap state")
}
