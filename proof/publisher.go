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
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
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
	socket          *zmq.Socket
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
	privateKey, err := zmqutil.ReadKeyFile(configuration.PrivateKey)
	if nil != err {
		log.Errorf("read private key file: %q  error: %v", configuration.PrivateKey, err)
		return err
	}
	publicKey, err := zmqutil.ReadKeyFile(configuration.PublicKey)
	if nil != err {
		log.Errorf("read public key file: %q  error: %v", configuration.PublicKey, err)
		return err
	}

	socket, err := zmq.NewSocket(zmq.PUB)
	if nil != err {
		return err
	}
	pub.socket = socket

	// ***** FIX THIS ****
	// this allows any client to connect
	zmq.AuthAllow(publisherZapDomain, "127.0.0.1/8")
	zmq.AuthCurveAdd(publisherZapDomain, zmq.CURVE_ALLOW_ANY)

	// domain is servers public key
	socket.SetCurveServer(1)
	//socket.SetCurvePublickey(publicKey)
	socket.SetCurveSecretkey(privateKey)
	log.Tracef("server public:  %q", publicKey)
	log.Tracef("server private: %q", privateKey)

	socket.SetZapDomain(publisherZapDomain)

	socket.SetIdentity(publicKey) // just use public key for identity

	// ***** FIX THIS ****
	// maybe need to change above line to specific keys later
	//   e.g. zmq.AuthCurveAdd(serverPublicKey, client1PublicKey)
	//        zmq.AuthCurveAdd(serverPublicKey, client2PublicKey)
	// perhaps as part of ConnectTo

	// // basic socket options
	// socket.SetIpv6(true) // ***** FIX THIS find fix for FreeBSD libzmq4 ****
	// socket.SetSndtimeo(SEND_TIMEOUT)
	// socket.SetLinger(LINGER_TIME)
	// socket.SetRouterMandatory(0)   // discard unroutable packets
	// socket.SetRouterHandover(true) // allow quick reconnect for a given public key
	// socket.SetImmediate(false)     // queue messages sent to disconnected peer
	for i, address := range configuration.Publish {
		bindTo, err := util.CanonicalIPandPort("tcp://", address)
		if nil != err {
			log.Errorf("publisher[%d]=%q  error: %v", i, address, err)
			return err
		}

		err = socket.Bind(bindTo)
		if nil != err {
			log.Errorf("publish[%d]=%q  error: %v", i, address, err)
			socket.Close()
			return err
		}
		log.Infof("publish on: %q", address)
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
	pub.socket.Close()
}

// process some items into a block and publish it
func (pub *publisher) process() {

	seenAsset := make(map[transactionrecord.AssetIndex]struct{})

	cursor := storage.Pool.VerifiedTransactions.NewFetchCursor()
	transactions, err := cursor.Fetch(blockrecord.MaximumTransactions)
	if nil != err {
		pub.log.Errorf("Error on Fetch: %v", err)
		return
	}

	if 0 == len(transactions) {
		pub.log.Info("verified pool is empty")
		return
	}

	txIds := make([]merkle.Digest, 1, len(transactions))
	txData := make([]byte, 0, 256*len(transactions))

	// to accumulate new assets
	assetIds := make([]transactionrecord.AssetIndex, 0, len(transactions))

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

	// first txid is the base
	txIds[0] = merkle.NewDigest(packedBase)

	for _, item := range transactions {
		unpacked, _, err := transactionrecord.Packed(item.Value).Unpack()
		if nil != err {
			pub.log.Criticalf("unpack error: %v", err)
			fault.PanicWithError("publisher extraction transactions", err)
		}

		// only issues and transfers are allowed here
		switch unpacked.(type) {
		case *transactionrecord.BitmarkIssue:
			issue := unpacked.(*transactionrecord.BitmarkIssue)

			if _, ok := seenAsset[issue.AssetIndex]; !ok {
				if !storage.Pool.Assets.Has(issue.AssetIndex[:]) {

					asset := storage.Pool.VerifiedAssets.Get(issue.AssetIndex[:])
					if nil == asset {
						pub.log.Criticalf("missing asset: %v", issue.AssetIndex)
						fault.Panicf("publisher missing asset: %v", issue.AssetIndex)
					}
					// add asset's transaction id to list
					txId := merkle.NewDigest(asset)
					txIds = append(txIds, txId)
					assetIds = append(assetIds, issue.AssetIndex)
					txData = append(txData, asset...)
				}
				seenAsset[issue.AssetIndex] = struct{}{}
			}

		case *transactionrecord.BitmarkTransfer:
			// ok

		default: // all other types cannot occure here
			pub.log.Criticalf("unxpected transaction: %v", unpacked)
			fault.Panicf("publisher unxpected transaction: %v", unpacked)
		}

		var digest merkle.Digest
		copy(digest[:], item.Key)
		txIds = append(txIds, digest)
		txData = append(txData, item.Value...)
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
	_, err = pub.socket.SendBytes(data, 0|zmq.DONTWAIT)
	fault.PanicIfError("publisher", err)
	time.Sleep(10 * time.Second)
}
