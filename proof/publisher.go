// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package proof

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/asset"
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/logger"
	zmq "github.com/pebbe/zmq4"
	"golang.org/x/crypto/ed25519"
	"io/ioutil"
	"time"
)

const (
	publishInterval    = 60 * time.Second
	publisherZapDomain = "publisher"
)

type publisher struct {
	log             *logger.L
	socket4         *zmq.Socket
	socket6         *zmq.Socket
	paymentCurrency currency.Currency
	paymentAddress  string
	owner           *account.Account
	privateKey      []byte
}

// initialise the publisher
func (pub *publisher) initialise(configuration *Configuration) error {

	log := logger.New("publisher")
	if nil == log {
		return fault.ErrInvalidLoggerChannel
	}
	pub.log = log

	log.Info("initialising…")

	_, err := fmt.Sscan(configuration.Currency, &pub.paymentCurrency)
	if nil != err {
		log.Errorf("currency: %q  error: %v", configuration.Currency, err)
		return err
	}
	pub.paymentAddress = configuration.Address

	if databytes, err := ioutil.ReadFile(configuration.SigningKey); err != nil {
		return err
	} else {
		rand := bytes.NewBuffer(databytes)
		publicKey, privateKey, err := ed25519.GenerateKey(rand)
		if nil != err {
			log.Errorf("public key generation  error: %v", err)
			return err
		}
		pub.owner = &account.Account{
			AccountInterface: &account.ED25519Account{
				Test:      true,
				PublicKey: publicKey,
			},
		}
		pub.privateKey = privateKey
	}

	// read the keys
	privateKey, err := zmqutil.ReadPrivateKeyFile(configuration.PrivateKey)
	if nil != err {
		log.Errorf("read private key file: %q  error: %v", configuration.PrivateKey, err)
		return err
	}
	publicKey, err := zmqutil.ReadPublicKeyFile(configuration.PublicKey)
	if nil != err {
		log.Errorf("read public key file: %q  error: %v", configuration.PublicKey, err)
		return err
	}
	log.Tracef("server public:  %x", publicKey)
	log.Tracef("server private: %x", privateKey)

	// create connections
	c, err := util.NewConnections(configuration.Publish)

	// allocate IPv4 and IPv6 sockets
	pub.socket4, pub.socket6, err = zmqutil.NewBind(log, zmq.PUB, publisherZapDomain, privateKey, publicKey, c)
	if nil != err {
		log.Errorf("bind error: %v", err)
		return err
	}

	return nil
}

// wait for new blocks or new payment items
// to ensure the queue integrity as heap is not thread-safe
func (pub *publisher) Run(args interface{}, shutdown <-chan struct{}) {

	log := pub.log

	log.Info("starting…")

loop:
	for {
		log.Info("waiting…")
		select {
		case <-shutdown:
			break loop
		case <-time.After(publishInterval):
			pub.process()
		}
	}
	if nil != pub.socket4 {
		pub.socket4.Close()
	}
	if nil != pub.socket6 {
		pub.socket6.Close()
	}
}

// process some items into a block and publish it
func (pub *publisher) process() {

	// only create new blocks if in normal mode
	if !mode.Is(mode.Normal) {
		return
	}

	seenAsset := make(map[transactionrecord.AssetIndex]struct{})

	txIds, transactions, totalByteCount, err := reservoir.Fetch(blockrecord.MaximumTransactions)
	if nil != err {
		pub.log.Errorf("Error on Fetch: %v", err)
		return
	}

	txCount := len(txIds)

	if 0 == txCount {
		pub.log.Info("verified pool is empty")
		return
	}

	// buffer to concatenate all transaction data
	txData := make([]byte, 0, totalByteCount)

	// to accumulate new assets
	assetIds := make([]transactionrecord.AssetIndex, 0, txCount)

	base := &transactionrecord.BaseData{
		Currency:       pub.paymentCurrency,
		PaymentAddress: pub.paymentAddress,
		Owner:          pub.owner,
		Nonce:          1234,
	}

	// sign the record and attach signature
	partiallyPackedBase, _ := base.Pack(pub.owner) // ignore error to get packed without signature
	signature := ed25519.Sign(pub.privateKey[:], partiallyPackedBase)
	base.Signature = signature[:]

	// re-pack to makesure signature is valid
	packedBase, err := base.Pack(pub.owner)
	if nil != err {
		pub.log.Criticalf("pack base error: %v", err)
		fault.PanicWithError("publisher packe base", err)
	}

	// first txId is the base
	txIds = append([]merkle.Digest{merkle.NewDigest(packedBase)}, txIds...)

	for _, item := range transactions {
		unpacked, _, err := transactionrecord.Packed(item).Unpack()
		if nil != err {
			pub.log.Criticalf("unpack error: %v", err)
			fault.PanicWithError("publisher extraction transactions", err)
		}

		// only issues and transfers are allowed here
		switch tx := unpacked.(type) {
		case *transactionrecord.BitmarkIssue:

			if _, ok := seenAsset[tx.AssetIndex]; !ok {
				if !storage.Pool.Assets.Has(tx.AssetIndex[:]) {

					packedAsset := asset.Get(tx.AssetIndex)
					if nil == packedAsset {
						pub.log.Criticalf("missing asset: %v", tx.AssetIndex)
						fault.Panicf("publisher missing asset: %v", tx.AssetIndex)
					}
					// add asset's transaction id to list
					txId := merkle.NewDigest(packedAsset)
					txIds = append(txIds, txId)
					assetIds = append(assetIds, tx.AssetIndex)
					txData = append(txData, packedAsset...)
				}
				seenAsset[tx.AssetIndex] = struct{}{}
			}

		case *transactionrecord.BitmarkTransfer:
			// ok

		default: // all other types cannot occur here
			pub.log.Criticalf("unxpected transaction: %v", unpacked)
			fault.Panicf("publisher unxpected transaction: %v", unpacked)
		}

		txData = append(txData, item...)
	}

	// build the tree of transaction IDs
	fullMerkleTree := merkle.FullMerkleTree(txIds)
	merkleRoot := fullMerkleTree[len(fullMerkleTree)-1]

	transactionCount := len(txIds)
	if transactionCount > blockrecord.MaximumTransactions {
		pub.log.Criticalf("too many transactions in block: %d", transactionCount)
		fault.Panicf("too many transactions in blok: %d", transactionCount)
	}

	// 64 bit nonce (8 bytes)
	randomBytes := make([]byte, 8)
	_, err = rand.Read(randomBytes)
	nonce := blockrecord.NonceType(binary.LittleEndian.Uint64(randomBytes))

	bits := difficulty.Current
	timestamp := uint64(time.Now().Unix())

	// PreviousBlock is all zero
	message := &PublishedItem{
		Job: "?", // set by enqueue
		Header: blockrecord.Header{
			Version:          blockrecord.Version,
			TransactionCount: uint16(transactionCount),
			MerkleRoot:       merkleRoot,
			Timestamp:        timestamp,
			Difficulty:       bits,
			Nonce:            nonce,
		},
		Base:     packedBase,
		TxIds:    txIds,
		AssetIds: assetIds,
	}
	message.Header.PreviousBlock, message.Header.Number = block.Get()

	// add job to the queue
	enqueueToJobQueue(message, txData)

	data, err := json.Marshal(message)
	fault.PanicIfError("JSON encode error: %v", err)

	pub.log.Infof("json to send: %s", data)

	// ***** FIX THIS: is the DONTWAIT flag needed or not?
	if nil != pub.socket4 {
		_, err = pub.socket4.SendBytes(data, 0|zmq.DONTWAIT)
		fault.PanicIfError("publisher 4", err)
	}
	if nil != pub.socket6 {
		_, err = pub.socket6.SendBytes(data, 0|zmq.DONTWAIT)
		fault.PanicIfError("publisher 6", err)
	}

	time.Sleep(10 * time.Second)
}
