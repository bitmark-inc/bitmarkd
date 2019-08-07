package storage

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

func TestPaymentStorageStoreBlock(t *testing.T) {
	db := PaymentStorage.Btc.DB()

	var height int32 = 100
	hash, err := chainhash.NewHashFromStr("c10ff375832ef8ca311669c17d87b435000f3542ee0aa36028c1902d9a40cd40")
	if err != nil {
		t.Fatalf("unable create hash: %s", err.Error())
	}

	store := PaymentStorage.Btc
	if err := store.StoreBlock(height, hash); err != nil {
		t.Fatalf("unable save hash into db: %s", err.Error())
	}

	heightKey := append([]byte{"h"[0]}, []byte(fmt.Sprintf("%08x", height))...)
	hashKey := append([]byte{"b"[0]}, hash.CloneBytes()...)

	hashValue, err := db.Get(heightKey, nil)

	if err != nil {
		t.Fatalf("unable to get value from db. error: %s", err.Error())
	}
	if !reflect.DeepEqual(hashKey[1:], hashValue) {
		t.Fatalf("hash value mis-match. except: %+v, actual: %+v", hashKey[1:], hashValue)
	}

	heightValue, err := db.Get(hashKey, nil)
	if err != nil {
		t.Fatalf("unable to get value from db. error: %s", err.Error())
	}
	if !reflect.DeepEqual(heightKey[1:], heightValue) {
		t.Fatalf("height value mis-match. except: %+v, actual: %+v", heightKey[1:], heightValue)
	}
}

func TestPaymentStorageGetHashAndHeight(t *testing.T) {
	var height int32 = 101
	hash, err := chainhash.NewHashFromStr("e18646bc3f9644422510b2ca54eb022c087896111e3ed93121456a384a411afc")
	if err != nil {
		t.Fatalf("unable create hash: %s", err.Error())
	}

	store := PaymentStorage.Btc
	if err := store.StoreBlock(height, hash); err != nil {
		t.Fatalf("unable save hash into db: %s", err.Error())
	}

	actualHash, err := store.GetHash(height)
	if err != nil {
		t.Fatalf("unable get hash: %s", err.Error())
	}

	if !reflect.DeepEqual(hash.CloneBytes(), actualHash.CloneBytes()) {
		t.Fatalf("height value mis-match. except: %+v, actual: %+v", hash.CloneBytes(), actualHash.CloneBytes())
	}

	actualHeight, err := store.GetHeight(hash)
	if err != nil {
		t.Fatalf("unable get height: %s", err.Error())
	}

	if !reflect.DeepEqual(height, actualHeight) {
		t.Fatalf("height value mis-match. except: %d, actual: %d", height, actualHeight)
	}
}

func TestPaymentStorageGetAndSetCheckpoint(t *testing.T) {
	var height int32 = 102
	hash, err := chainhash.NewHashFromStr("5158ab88474fa3b35c379a756a86e473236ddb92799f035b4c9df6c113003586")
	if err != nil {
		t.Fatalf("unable create hash: %s", err.Error())
	}

	store := PaymentStorage.Btc
	if err := store.StoreBlock(height, hash); err != nil {
		t.Fatalf("unable save hash into db: %s", err.Error())
	}

	if err := store.SetCheckpoint(height); err != nil {
		t.Fatalf("unable save checkpoint into db: %s", err.Error())
	}

	actualCheckpoint, err := store.GetCheckpoint()
	if err != nil {
		t.Fatalf("unable to get checkpoint. error: %s", err.Error())
	}

	if !reflect.DeepEqual(hash.CloneBytes(), actualCheckpoint.CloneBytes()) {
		t.Fatalf("hash value mis-match. expect: %+v, actual: %+v", hash.CloneBytes(), actualCheckpoint.CloneBytes())
	}
}

func TestPaymentStorageBlockReceipt(t *testing.T) {
	db := PaymentStorage.Btc.DB()

	var height int32 = 103

	store := PaymentStorage.Btc
	if err := store.SetBlockReceipt(height); err != nil {
		t.Fatalf("unable set block receipt. error: %s", err.Error())
	}

	receiptKey := append([]byte{"r"[0]}, []byte(fmt.Sprintf("%08x", height))...)
	exist, err := db.Has(receiptKey, nil)
	if err != nil {
		t.Fatalf("can not validate the receipt key from db. error: %s", err.Error())
	}

	if !exist {
		t.Fatalf("block receipt is not set to the database")
	}

	if !store.HasBlockReceipt(height) {
		t.Fatalf("unable get an existed block receipt")
	}
}

func TestPaymentStorageRollback(t *testing.T) {
	db := PaymentStorage.Btc.DB()

	var height1 int32 = 201
	var height2 int32 = 202
	var height3 int32 = 203
	hash1, err := chainhash.NewHashFromStr("a80ec41710e55c9867888117ffdd0004e3999cdb69b567196f688aa79972abc0")
	if err != nil {
		t.Fatalf("unable create hash: %s", err.Error())
	}

	hash2, err := chainhash.NewHashFromStr("bf117c0c23d7532c7c816cb5efe6db7e83df1909dca34a578c111817d5ba1b92")
	if err != nil {
		t.Fatalf("unable create hash: %s", err.Error())
	}

	hash3, err := chainhash.NewHashFromStr("fbd39ee206b31eea8a5576b5f97232df0f35a15ca7f63364183017191b48cb33")
	if err != nil {
		t.Fatalf("unable create hash: %s", err.Error())
	}

	store := PaymentStorage.Btc
	store.StoreBlock(height1, hash1)
	store.StoreBlock(height2, hash2)
	store.StoreBlock(height3, hash3)

	if err := store.RollbackTo(203, 201); err != nil {
		t.Fatalf("unable rollback blocks: %s", err.Error())
	}

	heightKey1 := append([]byte{"h"[0]}, []byte(fmt.Sprintf("%08x", height1))...)
	heightKey2 := append([]byte{"h"[0]}, []byte(fmt.Sprintf("%08x", height2))...)
	heightKey3 := append([]byte{"h"[0]}, []byte(fmt.Sprintf("%08x", height3))...)

	exist1, err := db.Has(heightKey1, nil)
	if err != nil {
		t.Fatalf("can not validate the height key from db. error: %s", err.Error())
	}
	if !exist1 {
		t.Fatal("height1 should exist")
	}

	exist2, err := db.Has(heightKey2, nil)
	if err != nil {
		t.Fatalf("can not validate the height key from db. error: %s", err.Error())
	}
	if exist2 {
		t.Fatal("height2 should not exist")
	}

	exist3, err := db.Has(heightKey3, nil)
	if err != nil {
		t.Fatalf("can not validate the height key from db. error: %s", err.Error())
	}
	if exist3 {
		t.Fatal("height3 should not exist")
	}
}
