// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package proof

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/blockheader"
	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/chain"
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/currency/bitcoin"
	"github.com/bitmark-inc/bitmarkd/currency/litecoin"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/logger"
	zmq "github.com/pebbe/zmq4"
	"golang.org/x/crypto/ed25519"
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
	log                *logger.L
	socket4            *zmq.Socket
	socket6            *zmq.Socket
	paymentAddress     map[currency.Currency]string
	owner              *account.Account
	privateKey         []byte
	internalHashEnable bool
}

// initialise the publisher
func (pub *publisher) initialise(configuration *Configuration) error {
	pub.internalHashEnable = configuration.InternalHashEnable
	log := logger.New("publisher")
	pub.log = log

	log.Info("initialising…")

	// set up payment address for each supported currency
	pub.paymentAddress = make(map[currency.Currency]string)
	for c, currencyAddress := range configuration.PaymentAddr {
		var paymentCurrency currency.Currency
		_, err := fmt.Sscan(c, &paymentCurrency)
		if err != nil {
			log.Errorf("currency: %q  error: %s", c, err)
			return err
		}

		switch paymentCurrency {
		case currency.Bitcoin:
			cType, _, err := bitcoin.ValidateAddress(currencyAddress)
			if err != nil {
				log.Errorf("validate bitcoin address error: %s", err)
				return err
			}
			switch cType {
			case bitcoin.Testnet, bitcoin.TestnetScript:
				if !mode.IsTesting() {
					err := fault.BitcoinAddressForWrongNetwork
					log.Errorf("validate bitcoin address error: %s", err)
					return err
				}
			case bitcoin.Livenet, bitcoin.LivenetScript:
				if mode.IsTesting() {
					err := fault.BitcoinAddressForWrongNetwork
					log.Errorf("validate bitcoin address error: %s", err)
					return err
				}
			default:
				return fault.BitcoinAddressIsNotSupported
			}
		case currency.Litecoin:
			cType, _, err := litecoin.ValidateAddress(currencyAddress)
			if err != nil {
				return err
			}
			switch cType {
			case litecoin.Testnet, litecoin.TestnetScript, litecoin.TestnetScript2:
				if !mode.IsTesting() {
					return fault.LitecoinAddressForWrongNetwork
				}
			case litecoin.Livenet, litecoin.LivenetScript, litecoin.LivenetScript2:
				if mode.IsTesting() {
					return fault.LitecoinAddressForWrongNetwork
				}
			default:
				return fault.LitecoinAddressIsNotSupported
			}

		default:
			log.Errorf("unsupported currency: %q", c)
			return fault.CurrencyIsNotSupportedByProofer
		}

		pub.paymentAddress[paymentCurrency] = currencyAddress
	}

	s := strings.TrimSpace(configuration.SigningKey)
	if strings.HasPrefix(s, taggedSeed) {
		privateKey, err := account.PrivateKeyFromBase58Seed(s[len(taggedSeed):])
		if err != nil {
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
		if err != nil {
			return err
		}
		pub.privateKey = privateKey.PrivateKeyBytes()
		pub.owner = privateKey.Account()
	} else {
		return fault.InvalidProofSigningKey
	}

	// when chain is local, use internal hasher
	if mode.ChainName() == chain.Local && pub.internalHashEnable {
		if err := newInternalHasherRequester(pub); err != nil {
			return err
		}
		return nil
	}

	// read the keys
	privateKey, err := zmqutil.ReadPrivateKey(configuration.PrivateKey)
	if err != nil {
		log.Errorf("read private key file: %q  error: %s", configuration.PrivateKey, err)
		return err
	}
	publicKey, err := zmqutil.ReadPublicKey(configuration.PublicKey)
	if err != nil {
		log.Errorf("read public key file: %q  error: %s", configuration.PublicKey, err)
		return err
	}
	log.Tracef("server public:  %x", publicKey)
	log.Tracef("server private: %x", privateKey)

	// create connections
	c, _ := util.NewConnections(configuration.Publish)

	// allocate IPv4 and IPv6 sockets
	pub.socket4, pub.socket6, err = zmqutil.NewBind(log, zmq.PUB, publisherZapDomain, privateKey, publicKey, c)
	if err != nil {
		log.Errorf("bind error: %s", err)
		return err
	}

	return nil
}

func newInternalHasherRequester(pub *publisher) error {
	var err error

	pub.socket4, err = zmq.NewSocket(internalHasherProtocol)
	if err != nil {
		return fmt.Errorf("create internal request hasher socket with error: %s", err)
	}

	err = pub.socket4.Bind(internalHasherRequest)
	if err != nil {
		return fmt.Errorf("bind internal hasher request socket with error: %s", err)
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
		log.Debug("waiting…")
		select {
		case <-shutdown:
			break loop
		case <-time.After(publishInterval): // timeout
			pub.process()
		}
	}
	if pub.socket4 != nil {
		pub.socket4.Close()
	}
	if pub.socket6 != nil {
		pub.socket6.Close()
	}
}

// process some items into a block and publish it
func (pub *publisher) process() {

	// only create new blocks if in normal mode
	if !mode.Is(mode.Normal) {
		return
	}

	// note: fetch one less tx because of foundation record
	pooledTxIds, transactions, err := reservoir.FetchVerified(blockrecord.MaximumTransactions - 1)
	if err != nil {
		pub.log.Errorf("Error on Fetch: %v", err)
		return
	}

	txCount := len(pooledTxIds)

	if txCount == 0 {
		pub.log.Info("verified pool is empty")
		return
	}

	// create record for each supported currency
	p := make(currency.Map)
	for c := currency.First; c <= currency.Last; c++ {
		p[c] = pub.paymentAddress[c]
	}

	blockFoundation := &transactionrecord.BlockFoundation{
		Version:  transactionrecord.FoundationVersion,
		Payments: p,
		Owner:    pub.owner,
		Nonce:    1234,
	}

	// sign the record and attach signature
	partiallyPacked, _ := blockFoundation.Pack(pub.owner) // ignore error to get packed without signature
	signature := ed25519.Sign(pub.privateKey, partiallyPacked)
	blockFoundation.Signature = signature

	// re-pack to makesure signature is valid
	packedBI, err := blockFoundation.Pack(pub.owner)
	if err != nil {
		pub.log.Criticalf("pack block foundation error: %s", err)
		logger.Panicf("publisher packed block foundation error: %s", err)
	}

	// the first two are base records
	txIds := make([]merkle.Digest, 1, len(pooledTxIds)*2) // allow room for inserted assets & allocate base
	txIds[0] = merkle.NewDigest(packedBI)
	txIds = append(txIds, pooledTxIds...)

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
	if err != nil {
		pub.log.Criticalf("random number generate with error: %s", err)
		logger.Panicf("random number generate with error: %s", err)
	}
	nonce := blockrecord.NonceType(binary.LittleEndian.Uint64(randomBytes))

	timestamp := uint64(time.Now().Unix())

	// PreviousBlock is all zero
	message := &PublishedItem{
		Job: "?", // set by enqueue
		Header: blockrecord.Header{
			Version:          blockrecord.Version,
			TransactionCount: uint16(transactionCount),
			MerkleRoot:       merkleRoot,
			Timestamp:        timestamp,
			Difficulty:       difficulty.Current,
			Nonce:            nonce,
		},
		TxZero: packedBI,
		TxIds:  txIds,
	}

	pub.log.Tracef("message: %v", message)

	message.Header.PreviousBlock, message.Header.Number = blockheader.GetNew()

	pub.log.Debugf("current difficulty: %f", message.Header.Difficulty.Value())
	if blockrecord.IsBlockToAdjustDifficulty(message.Header.Number, message.Header.Version) {
		newDifficulty, _ := blockrecord.DifficultyByPreviousTimespanAtBlock(message.Header.Number)

		diff := difficulty.New()
		diff.Set(newDifficulty)
		message.Header.Difficulty = diff

		pub.log.Debugf("difficulty adjust block %d, new difficulty: %f", message.Header.Number, newDifficulty)
	}

	// add job to the queue
	enqueueToJobQueue(message, transactions)

	data, err := json.Marshal(message)
	logger.PanicIfError("JSON encode error: %s", err)

	pub.log.Infof("json to send: %s", data)

	// ***** FIX THIS: is the DONTWAIT flag needed or not?
	if pub.socket4 != nil {
		_, err = pub.socket4.SendBytes(data, 0|zmq.DONTWAIT)
		logger.PanicIfError("publisher 4", err)
	}
	if pub.socket6 != nil {
		_, err = pub.socket6.SendBytes(data, 0|zmq.DONTWAIT)
		logger.PanicIfError("publisher 6", err)
	}
}
