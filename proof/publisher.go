// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package proof

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/asset"
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/currency/bitcoin"
	"github.com/bitmark-inc/bitmarkd/currency/litecoin"
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
	"strings"
	"time"
)

// tags for the signing key data
const (
	taggedSeed    = "SEED:"    // followed by base58 encoded seed as produced by desktop/cli client
	taggedPrivate = "PRIVATE:" // followed by 64 bytes of hex Ed25519 private key
)

const (
	publishBitmarkInterval = 60 * time.Second
	publishTestingInterval = 15 * time.Second
	publisherZapDomain     = "publisher"
)

type publisher struct {
	log            *logger.L
	socket4        *zmq.Socket
	socket6        *zmq.Socket
	paymentAddress map[currency.Currency]string
	owner          *account.Account
	privateKey     []byte
}

// initialise the publisher
func (pub *publisher) initialise(configuration *Configuration) error {

	log := logger.New("publisher")
	pub.log = log

	log.Info("initialising…")

	// set up payment address for each supported currency
	pub.paymentAddress = make(map[currency.Currency]string)
	for c, currencyAddress := range configuration.PaymentAddr {
		var paymentCurrency currency.Currency
		_, err := fmt.Sscan(c, &paymentCurrency)
		if nil != err {
			log.Errorf("currency: %q  error: %s", c, err)
			return err
		}

		switch paymentCurrency {
		case currency.Bitcoin:
			cType, _, err := bitcoin.ValidateAddress(currencyAddress)
			if nil != err {
				log.Errorf("validate bitcoin address error: %s", err)
				return err
			}
			switch cType {
			case bitcoin.Testnet, bitcoin.TestnetScript:
				if !mode.IsTesting() {
					err := fault.ErrBitcoinAddressForWrongNetwork
					log.Errorf("validate bitcoin address error: %s", err)
					return err
				}
			case bitcoin.Livenet, bitcoin.LivenetScript:
				if mode.IsTesting() {
					err := fault.ErrBitcoinAddressForWrongNetwork
					log.Errorf("validate bitcoin address error: %s", err)
					return err
				}
			default:
				return fault.ErrBitcoinAddressIsNotSupported
			}
		case currency.Litecoin:
			cType, _, err := litecoin.ValidateAddress(currencyAddress)
			if nil != err {
				return err
			}
			switch cType {
			case litecoin.Testnet, litecoin.TestnetScript:
				if !mode.IsTesting() {
					return fault.ErrLitecoinAddressForWrongNetwork
				}
			case litecoin.Livenet, litecoin.LivenetScript, litecoin.LivenetScript2:
				if mode.IsTesting() {
					return fault.ErrLitecoinAddressForWrongNetwork
				}
			default:
				return fault.ErrLitecoinAddressIsNotSupported
			}

		default:
			log.Errorf("unsupported currency: %q", c)
			return fault.ErrCurrencyIsNotSupportedByProofer
		}

		pub.paymentAddress[paymentCurrency] = currencyAddress
	}

	if databytes, err := ioutil.ReadFile(configuration.SigningKey); err != nil {
		return err
	} else {
		s := strings.TrimSpace(string(databytes))

		if strings.HasPrefix(s, taggedSeed) {
			privateKey, err := account.PrivateKeyFromBase58Seed(s[len(taggedSeed):])
			if nil != err {
				return err
			}
			pub.privateKey = privateKey.PrivateKeyBytes()
			pub.owner = privateKey.Account()
		} else if strings.HasPrefix(s, taggedPrivate) {
			b, err := hex.DecodeString(s[len(taggedPrivate):])
			if err != nil {
				return err
			}
			privateKey, err := account.PrivateKeyFromBytes(b)
			if nil != err {
				return err
			}
			pub.privateKey = privateKey.PrivateKeyBytes()
			pub.owner = privateKey.Account()
		} else {
			return fault.ErrInvalidProofSigningKey
		}
		rand := bytes.NewBuffer(databytes)
		publicKey, privateKey, err := ed25519.GenerateKey(rand)
		if nil != err {
			log.Errorf("public key generation  error: %s", err)
			return err
		}
		pub.owner = &account.Account{
			AccountInterface: &account.ED25519Account{
				Test:      mode.IsTesting(),
				PublicKey: publicKey,
			},
		}
		pub.privateKey = privateKey
	}

	// read the keys
	privateKey, err := zmqutil.ReadPrivateKeyFile(configuration.PrivateKey)
	if nil != err {
		log.Errorf("read private key file: %q  error: %s", configuration.PrivateKey, err)
		return err
	}
	publicKey, err := zmqutil.ReadPublicKeyFile(configuration.PublicKey)
	if nil != err {
		log.Errorf("read public key file: %q  error: %s", configuration.PublicKey, err)
		return err
	}
	log.Tracef("server public:  %x", publicKey)
	log.Tracef("server private: %x", privateKey)

	// create connections
	c, err := util.NewConnections(configuration.Publish)

	// allocate IPv4 and IPv6 sockets
	pub.socket4, pub.socket6, err = zmqutil.NewBind(log, zmq.PUB, publisherZapDomain, privateKey, publicKey, c)
	if nil != err {
		log.Errorf("bind error: %s", err)
		return err
	}

	return nil
}

// wait for new blocks or new payment items
// to ensure the queue integrity as heap is not thread-safe
func (pub *publisher) Run(args interface{}, shutdown <-chan struct{}) {

	log := pub.log

	log.Info("starting…")

	publishInterval := publishBitmarkInterval
	if mode.IsTesting() {
		publishInterval = publishTestingInterval
	}

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

	pooledTxIds, transactions, totalByteCount, err := reservoir.FetchVerified(blockrecord.MaximumTransactions)
	if nil != err {
		pub.log.Errorf("Error on Fetch: %v", err)
		return
	}

	txCount := len(pooledTxIds)

	if 0 == txCount {
		pub.log.Info("verified pool is empty")
		return
	}

	// buffer to concatenate all transaction data
	txData := make([]byte, 0, totalByteCount)

	// to accumulate new assets
	assetIds := make([]transactionrecord.AssetIndex, 0, txCount)

	// create record for each supported currency
	p := make(currency.Map)
	for c := currency.First; c <= currency.Last; c++ {
		p[c] = pub.paymentAddress[c]
	}

	blockIssue := &transactionrecord.BlockOwnerIssue{
		Version:  1,
		Payments: p,
		Owner:    pub.owner,
		Nonce:    1234,
	}

	// sign the record and attach signature
	partiallyPacked, _ := blockIssue.Pack(pub.owner) // ignore error to get packed without signature
	signature := ed25519.Sign(pub.privateKey[:], partiallyPacked)
	blockIssue.Signature = signature[:]

	// re-pack to makesure signature is valid
	packedBI, err := blockIssue.Pack(pub.owner)
	if nil != err {
		pub.log.Criticalf("pack base error: %s", err)
		logger.Panicf("publisher packed base error: %s", err)
	}

	// the first two are base records
	txIds := make([]merkle.Digest, 1, len(pooledTxIds)*2) // allow room for inserted assets & allocate base
	txIds[0] = merkle.NewDigest(packedBI)

	n := 0 // index for pooledTxIds
	for _, item := range transactions {
		unpacked, _, err := transactionrecord.Packed(item).Unpack()
		if nil != err {
			pub.log.Criticalf("unpack error: %s", err)
			logger.Panicf("publisher extraction transactions error: %s", err)
		}

		// only issues and transfers are allowed here
		switch tx := unpacked.(type) {
		case *transactionrecord.BitmarkIssue:

			if _, ok := seenAsset[tx.AssetIndex]; !ok {
				if !storage.Pool.Assets.Has(tx.AssetIndex[:]) {

					packedAsset := asset.Get(tx.AssetIndex)
					if nil == packedAsset {
						pub.log.Criticalf("missing asset: %v", tx.AssetIndex)
						logger.Panicf("publisher missing asset: %v", tx.AssetIndex)
					}
					// add asset's transaction id to list
					txId := merkle.NewDigest(packedAsset)
					txIds = append(txIds, txId)
					assetIds = append(assetIds, tx.AssetIndex)
					txData = append(txData, packedAsset...)
				}
				seenAsset[tx.AssetIndex] = struct{}{}
			}

		case *transactionrecord.BitmarkTransferUnratified, *transactionrecord.BitmarkTransferCountersigned:
			// ok

		default: // all other types cannot occur here
			pub.log.Criticalf("unxpected transaction: %v", unpacked)
			logger.Panicf("publisher unxpected transaction: %v", unpacked)
		}

		// concatenate items
		txIds = append(txIds, pooledTxIds[n])
		txData = append(txData, item...)
		n += 1
	}

	// build the tree of transaction IDs
	fullMerkleTree := merkle.FullMerkleTree(txIds)
	merkleRoot := fullMerkleTree[len(fullMerkleTree)-1]

	transactionCount := len(txIds)
	if transactionCount > blockrecord.MaximumTransactions {
		pub.log.Criticalf("too many transactions in block: %d", transactionCount)
		logger.Panicf("too many transactions in block: %d", transactionCount)
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
		TxZero:   packedBI,
		TxIds:    txIds,
		AssetIds: assetIds,
	}

	pub.log.Tracef("message: %v", message)

	message.Header.PreviousBlock, message.Header.Number = block.Get()

	// add job to the queue
	enqueueToJobQueue(message, txData)

	data, err := json.Marshal(message)
	logger.PanicIfError("JSON encode error: %s", err)

	pub.log.Infof("json to send: %s", data)

	// ***** FIX THIS: is the DONTWAIT flag needed or not?
	if nil != pub.socket4 {
		_, err = pub.socket4.SendBytes(data, 0|zmq.DONTWAIT)
		logger.PanicIfError("publisher 4", err)
	}
	if nil != pub.socket6 {
		_, err = pub.socket6.SendBytes(data, 0|zmq.DONTWAIT)
		logger.PanicIfError("publisher 6", err)
	}

	time.Sleep(10 * time.Second)
}
