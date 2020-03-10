// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package fixtures

import (
	"fmt"
	"os"

	zmq "github.com/pebbe/zmq4"

	"github.com/bitmark-inc/bitmarkd/util"

	"github.com/bitmark-inc/logger"
)

const (
	dir         = "testing"
	LogCategory = "testing"
)

var (
	Listener1  []byte
	Listener2  []byte
	PublicKey1 []byte
	PublicKey2 []byte
	PublicKey3 []byte
)

func init() {
	c, _ := util.NewConnection("127.0.0.1:1234")
	Listener1 = make([]byte, 0, 100)
	Listener1 = append(Listener1, c.Pack()...)

	c, _ = util.NewConnection("192.168.0.1:5678")
	Listener2 = make([]byte, 0, 100)
	Listener2 = append(Listener2, c.Pack()...)

	tmp, _, _ := zmq.NewCurveKeypair()
	PublicKey1 = []byte(zmq.Z85decode(tmp))

	tmp, _, _ = zmq.NewCurveKeypair()
	PublicKey2 = []byte(zmq.Z85decode(tmp))

	tmp, _, _ = zmq.NewCurveKeypair()
	PublicKey3 = []byte(zmq.Z85decode(tmp))
}

func SetupTestLogger() {
	removeFiles()
	_ = os.Mkdir(dir, 0700)

	logging := logger.Configuration{
		Directory: dir,
		File:      fmt.Sprintf("%s.log", LogCategory),
		Size:      1048576,
		Count:     10,
		Console:   false,
		Levels: map[string]string{
			logger.DefaultTag: "critical",
		},
	}

	// start logging
	_ = logger.Initialise(logging)
}

func TeardownTestLogger() {
	logger.Finalise()
	removeFiles()
}

func removeFiles() {
	err := os.RemoveAll(dir)
	if nil != err {
		fmt.Println("remove dir with error: ", err)
	}
}
