// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/rpc"
	"net/rpc/jsonrpc"
	"strconv"
	"strings"
	"time"

	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/blockheader"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/peer"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/logger"
)

// defaults
const (
	defaultCount = 10
	maximumCount = 100
)

// InternalConnection - type to allow rpc system to interface to http request
type InternalConnection struct {
	in  io.Reader
	out io.Writer
}

func (c *InternalConnection) Read(p []byte) (n int, err error) {
	return c.in.Read(p)
}
func (c *InternalConnection) Write(d []byte) (n int, err error) {
	return c.out.Write(d)
}
func (c *InternalConnection) Close() error {
	return nil
}

// the argument passed to the handlers
type httpHandler struct {
	log     *logger.L
	server  *rpc.Server
	start   time.Time
	version string
	allow   map[string][]*net.IPNet
}

// this matches anything not matched and returns error
func (s *httpHandler) root(w http.ResponseWriter, r *http.Request) {
	sendNotFound(w)
}

// performs a call to any normal RPC
func (s *httpHandler) rpc(w http.ResponseWriter, r *http.Request) {
	if http.MethodPost != r.Method {
		sendMethodNotAllowed(w)
		return
	}

	server := s.server

	connectionCount.Increment()
	defer connectionCount.Decrement()

	serverCodec := jsonrpc.NewServerCodec(&InternalConnection{in: r.Body, out: w})
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	err := server.ServeRequest(serverCodec)
	if nil != err {
		sendInternalServerError(w)
		return
	}
}

// check if remote address is allowed
func (s *httpHandler) isAllowed(api string, r *http.Request) bool {
	last := strings.LastIndex(r.RemoteAddr, ":")
	if last <= 0 {
		return false
	}

	cidr, ok := s.allow[api]
	if !ok {
		return false
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if nil != err {
		return false
	}

	addr := net.ParseIP(host)
	if nil == addr {
		return false
	}

	for _, n := range cidr {
		if n.Contains(addr) {
			return true
		}
	}

	return false
}

// to allow a GET for the same response and Node.Info RPC
func (s *httpHandler) details(w http.ResponseWriter, r *http.Request) {
	if http.MethodGet != r.Method {
		sendMethodNotAllowed(w)
		return
	}

	if !s.isAllowed("details", r) {
		s.log.Warnf("Deny access: %q", r.RemoteAddr)
		sendForbidden(w)
		return
	}

	connectionCount.Increment()
	defer connectionCount.Decrement()

	type lrCount struct {
		Local  uint64 `json:"local"`
		Remote uint64 `json:"remote"`
	}

	type blockInfo struct {
		LRCount lrCount `json:"count"`
		Hash    string  `json:"hash"`
	}

	type peerCounts struct {
		Incoming uint64 `json:"incoming"`
		Outgoing uint64 `json:"outgoing"`
	}

	type theReply struct {
		Chain               string     `json:"chain"`
		Mode                string     `json:"mode"`
		Block               blockInfo  `json:"block"`
		RPCs                uint64     `json:"rpcs"`
		Peers               peerCounts `json:"peers"`
		TransactionCounters Counters   `json:"transactionCounters"`
		Difficulty          float64    `json:"difficulty"`
		Hashrate            float64    `json:"hashrate,omitempty"`
		Version             string     `json:"version"`
		Uptime              string     `json:"uptime"`
		PublicKey           string     `json:"publicKey"`
	}

	reply := theReply{
		Chain: mode.ChainName(),
		Mode:  mode.String(),
		Block: blockInfo{
			LRCount: lrCount{
				Local:  blockheader.Height(),
				Remote: peer.BlockHeight(),
			},
			Hash: block.LastBlockHash(),
		},
		RPCs: connectionCount.Uint64(),
		// Miners : mine.ConnectionCount(),
		Difficulty: difficulty.Current.Value(),
		Hashrate:   difficulty.Hashrate(),
		Version:    s.version,
		Uptime:     time.Since(s.start).String(),
		PublicKey:  hex.EncodeToString(peer.PublicKey()),
	}

	reply.Peers.Incoming, reply.Peers.Outgoing = peer.GetCounts()
	reply.TransactionCounters.Pending, reply.TransactionCounters.Verified = reservoir.ReadCounters()

	sendReply(w, reply)
}

func (s *httpHandler) connections(w http.ResponseWriter, r *http.Request) {
	if http.MethodGet != r.Method {
		sendMethodNotAllowed(w)
		return
	}

	if !s.isAllowed("connections", r) {
		s.log.Warnf("Deny access: %q", r.RemoteAddr)
		sendForbidden(w)
		return
	}

	connectionCount.Increment()
	defer connectionCount.Decrement()

	type reply struct {
		ConnectedTo []*zmqutil.Connected `json:"connectedTo"`
	}

	var info reply

	info.ConnectedTo = peer.FetchConnectors()

	sendReply(w, info)
}

// to output peer data
type entry struct {
	PublicKey string    `json:"publicKey"`
	Listeners []string  `json:"listeners"`
	Timestamp time.Time `json:"timestamp"`
}

// GET to find data on all peers seen in the announcer
// (restricted to local_allow)
//
// query parameters:
//   public_key=<64-hex-characters>   [32 byte public key in hex]
//   count=<int>                      [1..100  default: 10]
func (s *httpHandler) peers(w http.ResponseWriter, r *http.Request) {
	if http.MethodGet != r.Method {
		sendMethodNotAllowed(w)
		return
	}

	if !s.isAllowed("peers", r) {
		s.log.Warnf("Deny access: %q", r.RemoteAddr)
		sendForbidden(w)
		return
	}

	connectionCount.Increment()
	defer connectionCount.Decrement()

	r.ParseForm()

	// public_key parsing
	startkey := []byte{}
	k, err := hex.DecodeString(r.Form.Get("public_key"))
	if nil == err && 32 == len(k) {
		startkey = k
	}

	// count parsing
	count := defaultCount
	n, err := strconv.Atoi(r.Form.Get("count"))
	if nil == err && n >= 1 && n <= maximumCount {
		count = n
	}

	peers := make([]entry, 0, count)

item_loop:
	for i := 0; i < count; i += 1 {
		publicKey, listeners, timestamp, err := announce.GetNext(startkey)
		if nil != err {
			sendInternalServerError(w)
			return
		}
		if bytes.Compare(publicKey, startkey) <= 0 {
			break item_loop
		}
		startkey = publicKey

		p := hex.EncodeToString(publicKey)

		lPack := util.PackedConnection(listeners)
		lc := make([]string, 0, 2)
	lc_loop:
		for {
			conn, n := lPack.Unpack()
			if nil == conn {
				break lc_loop
			}
			lc = append(lc, conn.String())
			lPack = lPack[n:]
		}

		peers = append(peers, entry{
			PublicKey: p,
			Listeners: lc,
			Timestamp: timestamp,
		})
	}

	sendReply(w, peers)
}

// send an JSON encoded reply
func sendReply(w http.ResponseWriter, data interface{}) {
	text, err := json.Marshal(data)
	if nil != err {
		sendInternalServerError(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	w.Write(text)
}

// selected errors as required above
func sendNotFound(w http.ResponseWriter) {
	sendError(w, "not found", http.StatusNotFound)
}
func sendMethodNotAllowed(w http.ResponseWriter) {
	sendError(w, "method not allowed", http.StatusMethodNotAllowed)
}
func sendForbidden(w http.ResponseWriter) {
	sendError(w, "forbidden", http.StatusForbidden)
}
func sendInternalServerError(w http.ResponseWriter) {
	sendError(w, "internal server error", http.StatusInternalServerError)
}

// to compose JSON error messages
type eType struct {
	Code  int    `json:"code"`
	Error string `json:"error"`
}

// output an error with a JSON body
func sendError(w http.ResponseWriter, message string, code int) {
	text, err := json.Marshal(eType{
		Code:  code,
		Error: message,
	})
	if nil != err {
		// manually composed error just incase JSON fails
		http.Error(w, `{"code":500,"error":"Internal Server Error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	w.Write(text)
}
