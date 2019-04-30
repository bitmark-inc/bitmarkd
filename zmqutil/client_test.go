// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package zmqutil

import (
	"crypto/rand"
	"testing"

	zmq "github.com/pebbe/zmq4"

	"github.com/bitmark-inc/bitmarkd/util"
)

const (
	defaultAddress = "127.0.0.1:9876"
	defaultChain   = "test"
	defaultTimeout = 0
)

func setupTestClient() *Client {
	publicKey := make([]byte, publicKeySize)
	privateKey := make([]byte, privateKeySize)
	_, _ = rand.Read(publicKey)
	_, _ = rand.Read(privateKey)
	client, _ := NewClient(zmq.SUB, privateKey, publicKey, defaultTimeout)
	return client
}

func teardownTestClient(c *Client) {
	_ = c.Close()
}

func TestGetSocket(t *testing.T) {
	client := setupTestClient()
	defer teardownTestClient(client)

	address, _ := util.NewConnection(defaultAddress)
	serverKey := make([]byte, publicKeySize)
	_, _ = rand.Read(serverKey)
	_ = client.Connect(address, serverKey, defaultChain)

	actual := client.GetSocket()
	expected := client.socket
	if actual != expected {
		t.Errorf("cannot get socket, expect %v but get %v",
			expected, actual)
	}
}
