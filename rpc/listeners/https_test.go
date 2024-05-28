// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package listeners_test

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/bitmarkd/rpc/certificate"
	"github.com/bitmark-inc/bitmarkd/rpc/fixtures"
	"github.com/bitmark-inc/bitmarkd/rpc/listeners"
	"github.com/bitmark-inc/logger"
)

type testHandler struct{}

func (h testHandler) RPC(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte("RPC"))
}

func (h testHandler) Details(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte("Details"))
}

func (h testHandler) Connections(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte("Connections"))
}

func (h testHandler) Peers(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte("Peers"))
}

func (h testHandler) Root(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte("Root"))
}

func (h testHandler) SetAllow(_ map[string][]*net.IPNet) {}

var client *http.Client

func init() {
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // ignore certificate verification

	client = &http.Client{
		Transport: customTransport,
	}
}

func setup(t *testing.T) (int, listeners.Listener) {
	allow := "127.0.0.1/32"
	port := rand.Intn(30000) + 30000

	listen := fmt.Sprintf("127.0.0.1:%d", port)
	conf := listeners.HTTPSConfiguration{
		MaximumConnections: 5,
		Listen:             []string{listen},
		Certificate:        "",
		PrivateKey:         "",
		Allow: map[string][]string{
			"details":     {allow},
			"connections": {allow},
			"rpc":         {allow},
			"root":        {allow},
			"peers":       {allow},
		},
	}

	wd, _ := os.Getwd()
	fixturePath := path.Join(filepath.Dir(wd), "fixtures")

	tlsConf, _, err := certificate.Get(
		logger.New(fixtures.LogCategory),
		"test",
		fixtures.Certificate(fixturePath),
		fixtures.Key(fixturePath),
	)
	if err != nil {
		t.Error("get certificate with error: ", err)
		t.FailNow()
	}

	h, err := listeners.NewHTTPS(
		&conf,
		logger.New(fixtures.LogCategory),
		tlsConf,
		testHandler{},
	)
	if err != nil {
		t.Error("NewHTTPS with error: ", err)
		t.FailNow()
	}

	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	return port, h
}

func TestHttpsListenerServeRPC(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	port, h := setup(t)

	err := h.Serve()
	assert.Nil(t, err, "wrong Serve")

	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	time.Sleep(time.Millisecond) // make sure server is ready
	url := fmt.Sprintf("https://127.0.0.1:%d/bitmarkd/", port)
	resp, err := client.Get(url + "rpc")
	if err != nil {
		t.Error("client get with error: ", err)
		t.FailNow()
	}
	defer resp.Body.Close()

	content, _ := ioutil.ReadAll(resp.Body)
	assert.Equal(t, "RPC", string(content), "wrong RPC call")
}

func TestHttpsListenerServeDetails(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	port, h := setup(t)

	err := h.Serve()
	assert.Nil(t, err, "wrong Serve")

	time.Sleep(time.Millisecond)
	url := fmt.Sprintf("https://127.0.0.1:%d/bitmarkd/", port)
	resp, err := client.Get(url + "details")
	if err != nil {
		t.Error("client get with error: ", err)
		t.FailNow()
	}
	defer resp.Body.Close()

	content, _ := ioutil.ReadAll(resp.Body)
	assert.Equal(t, "Details", string(content), "wrong Details call")
}

func TestHttpsListenerServePeers(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	port, h := setup(t)

	err := h.Serve()
	assert.Nil(t, err, "wrong Serve")

	time.Sleep(time.Millisecond)
	url := fmt.Sprintf("https://127.0.0.1:%d/bitmarkd/", port)
	resp, err := client.Get(url + "peers")
	if err != nil {
		t.Error("client get with error: ", err)
		t.FailNow()
	}
	defer resp.Body.Close()

	content, _ := ioutil.ReadAll(resp.Body)
	assert.Equal(t, "Peers", string(content), "wrong Peers call")
}

func TestHttpsListenerServeConnections(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	port, h := setup(t)

	err := h.Serve()
	assert.Nil(t, err, "wrong Serve")

	time.Sleep(time.Millisecond)
	url := fmt.Sprintf("https://127.0.0.1:%d/bitmarkd/", port)
	resp, err := client.Get(url + "connections")
	if err != nil {
		t.Error("client get with error: ", err)
		t.FailNow()
	}
	defer resp.Body.Close()

	content, _ := ioutil.ReadAll(resp.Body)
	assert.Equal(t, "Connections", string(content), "wrong Connections call")
}

func TestHttpsListenerServeRoot(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	port, h := setup(t)

	err := h.Serve()
	assert.Nil(t, err, "wrong Serve")

	time.Sleep(time.Millisecond)
	url := fmt.Sprintf("https://127.0.0.1:%d/bitmarkd/", port)
	resp, err := client.Get(url)
	if err != nil {
		t.Error("client get with error: ", err)
		t.FailNow()
	}
	defer resp.Body.Close()

	content, _ := ioutil.ReadAll(resp.Body)
	assert.Equal(t, "Root", string(content), "wrong Root call")
}
