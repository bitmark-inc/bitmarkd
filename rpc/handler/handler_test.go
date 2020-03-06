// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package handler_test

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"net/rpc"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/logger"

	"github.com/bitmark-inc/bitmarkd/rpc/fixtures"
	"github.com/bitmark-inc/bitmarkd/rpc/handler"
)

const (
	notAllowed      = "method not allowed"
	tooManyRequests = "Too Many Requests"
)

type eResp struct {
	Code  int    `json:"code"`
	Error string `json:"error"`
}

type jResp struct {
	ID     int   `json:"id"`
	Result int   `json:"result"`
	Error  error `json:"error"`
}

type jReq struct {
	ID     int      `json:"id"`
	Method string   `json:"method"`
	Params []AddArg `json:"params"`
}

type Add struct{}
type AddArg struct {
	A int `json:"A"`
	B int `json:"B"`
}

func (a Add) Add(arg *AddArg, reply *int) error {
	*reply = arg.A + arg.B
	return nil
}

func TestRoot(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	s := rpc.NewServer()

	h := handler.New(
		logger.New(fixtures.LogCategory),
		s,
		time.Now(),
		"1.0",
		uint64(5),
	)

	req := httptest.NewRequest("GET", "http://not.found", nil)
	w := httptest.NewRecorder()
	h.Root(w, req)

	resp := w.Result()
	var j eResp
	_ = json.NewDecoder(resp.Body).Decode(&j)

	assert.Equal(t, "not found", j.Error, "wrong response")
	assert.Equal(t, http.StatusNotFound, j.Code, "wrong http code")
}

func TestRPC(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	s := rpc.NewServer()
	a := Add{}
	_ = s.Register(a)

	h := handler.New(
		logger.New(fixtures.LogCategory),
		s,
		time.Now(),
		"1.0",
		uint64(5),
	)

	add := AddArg{
		A: 1,
		B: 2,
	}

	arg := jReq{
		ID:     5,
		Method: "Add.Add",
		Params: []AddArg{add},
	}
	data, _ := json.Marshal(arg)

	req := httptest.NewRequest("POST", "http://not.exist", bytes.NewReader(data))
	w := httptest.NewRecorder()
	h.RPC(w, req)

	resp := w.Result()
	var j jResp
	_ = json.NewDecoder(resp.Body).Decode(&j)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "wrong status code")
	assert.Equal(t, add.A+add.B, j.Result, "wrong result")
	assert.Nil(t, j.Error, "wrong error")
}

func TestRPCWhenWrongHTTPMethod(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	s := rpc.NewServer()
	a := Add{}
	_ = s.Register(a)

	h := handler.New(
		logger.New(fixtures.LogCategory),
		s,
		time.Now(),
		"1.0",
		uint64(5),
	)

	req := httptest.NewRequest("GET", "http://not.exist", nil)
	w := httptest.NewRecorder()
	h.RPC(w, req)

	resp := w.Result()
	var j eResp
	_ = json.NewDecoder(resp.Body).Decode(&j)
	assert.Equal(t, notAllowed, j.Error, "wrong method")
}

func TestRPCWhenTooManyConnections(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	s := rpc.NewServer()
	a := Add{}
	_ = s.Register(a)

	h := handler.New(
		logger.New(fixtures.LogCategory),
		s,
		time.Now(),
		"1.0",
		uint64(0),
	)

	req := httptest.NewRequest("POST", "http://not.exist", nil)
	w := httptest.NewRecorder()
	h.RPC(w, req)

	resp := w.Result()
	var j eResp
	_ = json.NewDecoder(resp.Body).Decode(&j)
	assert.Equal(t, tooManyRequests, j.Error, "wrong method")
}

func TestRPCWhenServeError(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	s := rpc.NewServer()

	h := handler.New(
		logger.New(fixtures.LogCategory),
		s,
		time.Now(),
		"1.0",
		uint64(5),
	)

	arg := jReq{}
	data, _ := json.Marshal(arg)

	req := httptest.NewRequest("POST", "http://not.exist", bytes.NewReader(data))
	w := httptest.NewRecorder()
	h.RPC(w, req)

	resp := w.Result()
	b, _ := ioutil.ReadAll(resp.Body)
	assert.Contains(t, string(b), "internal server error", "wrong response")
}

func TestPeers(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	s := rpc.NewServer()

	h := handler.New(
		logger.New(fixtures.LogCategory),
		s,
		time.Now(),
		"1.0",
		uint64(5),
	)

	allow := make(map[string][]*net.IPNet)
	_, ipNet, _ := net.ParseCIDR("192.0.2.1/32")
	allow["peers"] = []*net.IPNet{ipNet}
	h.SetAllow(allow)

	params := make(url.Values)
	params["peerid"] = []string{"12345"}

	req := httptest.NewRequest("GET", "http://test.com?"+params.Encode(), nil)
	w := httptest.NewRecorder()

	h.Peers(w, req)

	resp := w.Result()
	var j jResp
	_ = json.NewDecoder(resp.Body).Decode(&j)
	assert.Equal(t, 0, j.Result, "wrong result")
}

func TestPeersWhenWrongHTTPMethod(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	s := rpc.NewServer()

	h := handler.New(
		logger.New(fixtures.LogCategory),
		s,
		time.Now(),
		"1.0",
		uint64(5),
	)

	req := httptest.NewRequest("POST", "http://not.exist", nil)
	w := httptest.NewRecorder()
	h.Peers(w, req)

	resp := w.Result()
	var j eResp
	_ = json.NewDecoder(resp.Body).Decode(&j)
	assert.Equal(t, notAllowed, j.Error, "wrong method")
}

func TestPeersWhenNotAllow(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	s := rpc.NewServer()

	h := handler.New(
		logger.New(fixtures.LogCategory),
		s,
		time.Now(),
		"1.0",
		uint64(5),
	)

	req := httptest.NewRequest("GET", "http://test.com", nil)
	w := httptest.NewRecorder()

	h.Peers(w, req)

	resp := w.Result()
	var j eResp
	_ = json.NewDecoder(resp.Body).Decode(&j)
	assert.Equal(t, "forbidden", j.Error, "wrong not allow")
}

func TestPeersWhenTooManyConnections(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	s := rpc.NewServer()

	h := handler.New(
		logger.New(fixtures.LogCategory),
		s,
		time.Now(),
		"1.0",
		uint64(0),
	)

	allow := make(map[string][]*net.IPNet)
	_, ipNet, _ := net.ParseCIDR("192.0.2.1/32")
	allow["peers"] = []*net.IPNet{ipNet}
	h.SetAllow(allow)

	req := httptest.NewRequest("GET", "http://not.exist", nil)
	w := httptest.NewRecorder()
	h.Peers(w, req)

	resp := w.Result()
	var j eResp
	_ = json.NewDecoder(resp.Body).Decode(&j)
	assert.Equal(t, tooManyRequests, j.Error, "wrong method")
}

func TestConnections(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	s := rpc.NewServer()

	h := handler.New(
		logger.New(fixtures.LogCategory),
		s,
		time.Now(),
		"1.0",
		uint64(1),
	)

	allow := make(map[string][]*net.IPNet)
	_, ipNet, _ := net.ParseCIDR("192.0.2.1/32")
	allow["connections"] = []*net.IPNet{ipNet}
	h.SetAllow(allow)

	req := httptest.NewRequest("GET", "http://test.com", nil)
	w := httptest.NewRecorder()
	h.Connections(w, req)

	resp := w.Result()
	b, _ := ioutil.ReadAll(resp.Body)
	assert.Contains(t, string(b), "connectedTo", "wrong response")
}

func TestConnectionWhenWrongHTTPMethod(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	s := rpc.NewServer()

	h := handler.New(
		logger.New(fixtures.LogCategory),
		s,
		time.Now(),
		"1.0",
		uint64(5),
	)

	req := httptest.NewRequest("POST", "http://not.exist", nil)
	w := httptest.NewRecorder()
	h.Connections(w, req)

	resp := w.Result()
	var j eResp
	_ = json.NewDecoder(resp.Body).Decode(&j)
	assert.Equal(t, notAllowed, j.Error, "wrong method")
}

func TestConnectionsWhenNotAllow(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	s := rpc.NewServer()

	h := handler.New(
		logger.New(fixtures.LogCategory),
		s,
		time.Now(),
		"1.0",
		uint64(5),
	)

	req := httptest.NewRequest("GET", "http://test.com", nil)
	w := httptest.NewRecorder()

	h.Connections(w, req)

	resp := w.Result()
	var j eResp
	_ = json.NewDecoder(resp.Body).Decode(&j)
	assert.Equal(t, "forbidden", j.Error, "wrong not allow")
}

func TestConnectionsWhenTooManyConnections(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	s := rpc.NewServer()

	h := handler.New(
		logger.New(fixtures.LogCategory),
		s,
		time.Now(),
		"1.0",
		uint64(0),
	)

	allow := make(map[string][]*net.IPNet)
	_, ipNet, _ := net.ParseCIDR("192.0.2.1/32")
	allow["connections"] = []*net.IPNet{ipNet}
	h.SetAllow(allow)

	req := httptest.NewRequest("GET", "http://not.exist", nil)
	w := httptest.NewRecorder()
	h.Connections(w, req)

	resp := w.Result()
	var j eResp
	_ = json.NewDecoder(resp.Body).Decode(&j)
	assert.Equal(t, tooManyRequests, j.Error, "wrong method")
}

func TestDetails(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	s := rpc.NewServer()

	h := handler.New(
		logger.New(fixtures.LogCategory),
		s,
		time.Now(),
		"1.0",
		uint64(10),
	)

	allow := make(map[string][]*net.IPNet)
	_, ipNet, _ := net.ParseCIDR("192.0.2.1/32")
	allow["details"] = []*net.IPNet{ipNet}
	h.SetAllow(allow)

	req := httptest.NewRequest("GET", "http://test.com", nil)
	w := httptest.NewRecorder()

	h.Details(w, req)

	resp := w.Result()
	b, _ := ioutil.ReadAll(resp.Body)
	assert.Contains(t, string(b), "Stopped", "wrong response")
}

func TestDetailsWhenWrongHTTPMethod(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	s := rpc.NewServer()

	h := handler.New(
		logger.New(fixtures.LogCategory),
		s,
		time.Now(),
		"1.0",
		uint64(5),
	)

	req := httptest.NewRequest("POST", "http://not.exist", nil)
	w := httptest.NewRecorder()
	h.Details(w, req)

	resp := w.Result()
	var j eResp
	_ = json.NewDecoder(resp.Body).Decode(&j)
	assert.Equal(t, notAllowed, j.Error, "wrong method")
}

func TestDetailsWhenNotAllow(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	s := rpc.NewServer()

	h := handler.New(
		logger.New(fixtures.LogCategory),
		s,
		time.Now(),
		"1.0",
		uint64(5),
	)

	req := httptest.NewRequest("GET", "http://test.com", nil)
	w := httptest.NewRecorder()

	h.Details(w, req)

	resp := w.Result()
	var j eResp
	_ = json.NewDecoder(resp.Body).Decode(&j)
	assert.Equal(t, "forbidden", j.Error, "wrong not allow")
}

func TestDetailsWhenTooManyConnections(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	s := rpc.NewServer()

	h := handler.New(
		logger.New(fixtures.LogCategory),
		s,
		time.Now(),
		"1.0",
		uint64(0),
	)

	allow := make(map[string][]*net.IPNet)
	_, ipNet, _ := net.ParseCIDR("192.0.2.1/32")
	allow["details"] = []*net.IPNet{ipNet}
	h.SetAllow(allow)

	req := httptest.NewRequest("GET", "http://not.exist", nil)
	w := httptest.NewRecorder()
	h.Details(w, req)

	resp := w.Result()
	var j eResp
	_ = json.NewDecoder(resp.Body).Decode(&j)
	assert.Equal(t, tooManyRequests, j.Error, "wrong method")
}
