// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"

	zmq "github.com/pebbe/zmq4"

	"github.com/bitmark-inc/logger"
)

// Submission - receive submission and reply to it
func Submission(bindTo string, publicKey []byte, privateKey []byte, log *logger.L) error {

	log.Info("starting…")

	socket, err := zmq.NewSocket(zmq.REP)
	if err != nil {
		return err
	}

	log.Infof("bind to: %q", bindTo)

	// this allows any client to connect
	//zmq.AuthAllow("submit", "127.0.0.1/8")
	zmq.AuthCurveAdd("submit", zmq.CURVE_ALLOW_ANY)

	// domain is servers public key
	socket.SetCurveServer(1)
	//socket.SetCurvePublickey(string(publicKey))
	socket.SetCurveSecretkey(string(privateKey))
	log.Infof("server public:  %x", publicKey)
	log.Infof("server private: %x", privateKey)
	socket.SetZapDomain("submit")

	socket.SetIdentity(string(publicKey)) // just use public key for identity

	// // basic socket options
	// //socket.SetIpv6(true)  // ***** FIX THIS find fix for FreeBSD libzmq4 ****
	// socket.SetSndtimeo(SEND_TIMEOUT)
	// socket.SetLinger(LINGER_TIME)
	// socket.SetRouterMandatory(0)   // discard unroutable packets
	// socket.SetRouterHandover(true) // allow quick reconnect for a given public key
	// socket.SetImmediate(false)     // queue messages sent to disconnected peer

	// // servers identity
	// socket.SetIdentity(publicKey) // just use public key for identity

	err = socket.Bind(bindTo)
	if err != nil {
		socket.Close()
		return err
	}

	// background process
	go func() {
		defer socket.Close()

		n := 0
	receiving:
		for {
			n += 1

			data, err := socket.RecvMessage(0)
			if err != nil {
				log.Errorf("JSON encode error: %s", err)
				break receiving
			}

			log.Infof("received message: %q", data)

			// var request interface{}
			// err = json.Unmarshal([]byte(data), &request)
			// if err != nil {
			// 	log.Errorf("JSON decode error: %s", err)
			// 	continue receiving
			// }

			// log.Infof("received message: %v", request)

			response := struct {
				N  int
				OK bool
			}{
				N:  n,
				OK: true,
			}

			result, err := json.Marshal(response)
			if err != nil {
				log.Errorf("JSON encode error: %s", err)
				continue receiving
			}
			log.Infof("json to send: %s\n", result)

			// if _, err := socket.Send(to, zmq.SNDMORE|zmq.DONTWAIT); err != nil {
			// 	return err
			// }
			// if _, err := socket.Send(command, zmq.SNDMORE|zmq.DONTWAIT); err != nil {
			// 	return err
			// }
			_, err = socket.SendBytes(result, 0|zmq.DONTWAIT)
			logger.PanicIfError("Submission", err)
		}
	}()
	return nil
}
