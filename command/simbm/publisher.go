// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"time"

	zmq "github.com/pebbe/zmq4"

	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/logger"
)

// PublishedItem - to send to proofer
type PublishedItem struct {
	Job    string
	Header blockrecord.Header
}

// Publish - setup the publisher
func Publish(bindTo string, publicKey []byte, privateKey []byte, log *logger.L) error {

	log.Info("starting…")

	socket, err := zmq.NewSocket(zmq.PUB)
	if err != nil {
		return err
	}

	log.Infof("bind to: %q", bindTo)

	// this allows any client to connect
	//zmq.AuthAllow("publisher", "127.0.0.1/8")
	zmq.AuthCurveAdd("publisher", zmq.CURVE_ALLOW_ANY)

	// domain is servers public key
	socket.SetCurveServer(1)
	//socket.SetCurvePublickey(string(publicKey))
	socket.SetCurveSecretkey(string(privateKey))
	log.Infof("server public:  %x", publicKey)
	log.Infof("server private: %x", privateKey)

	socket.SetZapDomain("publisher")

	socket.SetIdentity(string(publicKey)) // just use public key for identity

	// // basic socket options
	// //socket.SetIpv6(true)  // ***** FIX THIS find fix for FreeBSD libzmq4 ****
	// socket.SetSndtimeo(SEND_TIMEOUT)
	// socket.SetLinger(LINGER_TIME)
	// socket.SetRouterMandatory(0)   // discard unroutable packets
	// socket.SetRouterHandover(true) // allow quick reconnect for a given public key
	// socket.SetImmediate(false)     // queue messages sent to disconnected peer

	err = socket.Bind(bindTo)
	if err != nil {
		socket.Close()
		return err
	}

	// background process
	go func() {
		defer socket.Close()

		n := 0
		test := false
	loop:
		for {
			n += 1
			bits := difficulty.New()

			randomBytes := make([]byte, 8)
			_, err := rand.Read(randomBytes)
			nonce := blockrecord.NonceType(binary.LittleEndian.Uint64(randomBytes))

			liveMerkleRoot := merkle.Digest{
				0x63, 0x8c, 0x15, 0x9c, 0x1f, 0x11, 0x3f, 0x70,
				0xa9, 0x86, 0x6d, 0x9a, 0x9e, 0x52, 0xe9, 0xef,
				0xe9, 0xb9, 0x92, 0x08, 0x48, 0xad, 0x1d, 0xf3,
				0x48, 0x51, 0xbe, 0x8a, 0x56, 0x2a, 0x99, 0x8d,
			}
			testMerkleRoot := merkle.Digest{
				0xee, 0x07, 0xbb, 0xc3, 0xd7, 0x49, 0xe0, 0x7d,
				0x24, 0xb9, 0x0c, 0xd1, 0xec, 0x35, 0x14, 0x70,
				0x2e, 0x87, 0x85, 0x22, 0xda, 0xf7, 0x16, 0xc1,
				0x73, 0x24, 0xd6, 0x66, 0x69, 0x7b, 0x8a, 0x63,
			}

			var merkleRoot merkle.Digest
			var timestamp uint64
			job := "unknown"
			if test {
				job = "test"
				merkleRoot = testMerkleRoot
				timestamp = uint64(0x5478424b) // test
				test = false
			} else {
				job = "live"
				merkleRoot = liveMerkleRoot
				timestamp = uint64(0x56809ab7) // live
				test = true
			}

			// PreviousBlock is all zero
			message := PublishedItem{
				Job: job,
				Header: blockrecord.Header{
					Version:          1,
					TransactionCount: 1,
					Number:           1,
					MerkleRoot:       merkleRoot,
					Timestamp:        timestamp,
					Difficulty:       bits,
					Nonce:            nonce,
				},
			}

			data, err := json.Marshal(message)
			if err != nil {
				log.Errorf("JSON encode error: %s", err)
				continue loop
			}
			log.Infof("json to send: %s\n", data)

			_, err = socket.SendBytes(data, 0|zmq.DONTWAIT)
			logger.PanicIfError("publisher", err)
			time.Sleep(10 * time.Second)
		}
	}()
	return nil
}
