// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package handler

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
	"github.com/bitmark-inc/bitmarkd/counter"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/peer"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/bitmarkd/rpc/node"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/logger"
)

// defaults
const (
	defaultCount = 10
	maximumCount = 100
)

// InternalConnection - type to allow RPC system to interface to http request
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
type Handler interface {
	Peers(http.ResponseWriter, *http.Request)
	RPC(http.ResponseWriter, *http.Request)
	Details(http.ResponseWriter, *http.Request)
	Connections(http.ResponseWriter, *http.Request)
	Root(http.ResponseWriter, *http.Request)
	SetAllow(allow map[string][]*net.IPNet)
}

type handler struct {
	log                *logger.L
	server             *rpc.Server
	start              time.Time
	version            string
	allow              map[string][]*net.IPNet
	maximumConnections uint64
}

func New(
	log *logger.L,
	server *rpc.Server,
	start time.Time,
	version string,
	maximumConnections uint64,
) Handler {
	return &handler{
		log:                log,
		server:             server,
		start:              start,
		version:            version,
		maximumConnections: maximumConnections,
	}
}

func (h *handler) SetAllow(allow map[string][]*net.IPNet) {
	h.allow = allow
}

// global atomic connection counter
// all listening ports share this count
var connectionCountHTTPS counter.Counter

// this matches anything not matched and returns error
func (h *handler) Root(w http.ResponseWriter, _ *http.Request) {
	sendNotFound(w)
}

// performs a call to any normal RPC
func (h *handler) RPC(w http.ResponseWriter, r *http.Request) {
	if http.MethodPost != r.Method {
		sendMethodNotAllowed(w)
		return
	}

	server := h.server

	if connectionCountHTTPS.Increment() > h.maximumConnections {
		connectionCountHTTPS.Decrement()
		sendTooManyRequests(w)
		return
	}
	defer connectionCountHTTPS.Decrement()

	serverCodec := jsonrpc.NewServerCodec(&InternalConnection{in: r.Body, out: w})

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	err := server.ServeRequest(serverCodec)
	if err != nil {
		sendInternalServerError(w)
		return
	}
}

// check if remote address is allowed
func (h *handler) isAllowed(api string, r *http.Request) bool {
	last := strings.LastIndex(r.RemoteAddr, ":")
	if last <= 0 {
		return false
	}

	cidr, ok := h.allow[api]
	if !ok {
		return false
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return false
	}

	addr := net.ParseIP(host)
	if addr == nil {
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
func (h *handler) Details(w http.ResponseWriter, r *http.Request) {
	if http.MethodGet != r.Method {
		sendMethodNotAllowed(w)
		return
	}

	if !h.isAllowed("details", r) {
		h.log.Warnf("Deny access: %q", r.RemoteAddr)
		sendForbidden(w)
		return
	}

	if connectionCountHTTPS.Increment() > h.maximumConnections {
		connectionCountHTTPS.Decrement()
		sendTooManyRequests(w)
		return
	}
	defer connectionCountHTTPS.Decrement()

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
		Chain               string        `json:"chain"`
		Mode                string        `json:"mode"`
		Block               blockInfo     `json:"block"`
		RPCs                uint64        `json:"rpcs"`
		Peers               peerCounts    `json:"peers"`
		TransactionCounters node.Counters `json:"transactionCounters"`
		Difficulty          float64       `json:"difficulty"`
		Hashrate            float64       `json:"hashrate,omitempty"`
		Version             string        `json:"version"`
		Uptime              string        `json:"uptime"`
		PublicKey           string        `json:"publicKey"`
	}

	reply := theReply{
		Chain: mode.ChainName(),
		Mode:  mode.String(),
		Block: blockInfo{
			LRCount: lrCount{
				Local:  blockheader.Height(),
				Remote: peer.BlockHeight(),
			},
			Hash: block.LastBlockHash(storage.Pool.Blocks),
		},
		RPCs: connectionCountHTTPS.Uint64(),
		// Miners : mine.ConnectionCount(),
		Difficulty: difficulty.Current.Value(),
		Hashrate:   difficulty.Hashrate(),
		Version:    h.version,
		Uptime:     time.Since(h.start).String(),
		PublicKey:  hex.EncodeToString(peer.PublicKey()),
	}

	reply.Peers.Incoming, reply.Peers.Outgoing = peer.GetCounts()
	reply.TransactionCounters.Pending, reply.TransactionCounters.Verified = reservoir.ReadCounters()

	sendReply(w, reply)
}

func (h *handler) Connections(w http.ResponseWriter, r *http.Request) {
	if http.MethodGet != r.Method {
		sendMethodNotAllowed(w)
		return
	}

	if !h.isAllowed("connections", r) {
		h.log.Warnf("Deny access: %q", r.RemoteAddr)
		sendForbidden(w)
		return
	}

	if connectionCountHTTPS.Increment() > h.maximumConnections {
		connectionCountHTTPS.Decrement()
		sendTooManyRequests(w)
		return
	}
	defer connectionCountHTTPS.Decrement()

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
//
//	public_key=<64-hex-characters>   [32 byte public key in hex]
//	count=<int>                      [1..100  default: 10]
func (h *handler) Peers(w http.ResponseWriter, r *http.Request) {
	if http.MethodGet != r.Method {
		sendMethodNotAllowed(w)
		return
	}
	if !h.isAllowed("peers", r) {
		h.log.Warnf("Deny access: %q", r.RemoteAddr)
		sendForbidden(w)
		return
	}

	if connectionCountHTTPS.Increment() > h.maximumConnections {
		connectionCountHTTPS.Decrement()
		sendTooManyRequests(w)
		return
	}
	defer connectionCountHTTPS.Decrement()

	r.ParseForm()

	// public_key parsing
	startkey := []byte{}
	k, err := hex.DecodeString(r.Form.Get("public_key"))
	if err == nil && len(k) == 32 {
		startkey = k
	}

	// count parsing
	count := defaultCount
	n, err := strconv.Atoi(r.Form.Get("count"))
	if err == nil && n >= 1 && n <= maximumCount {
		count = n
	}
	peers := make([]entry, 0, count)

item_loop:
	for i := 0; i < count; i += 1 {
		publicKey, listeners, timestamp, err := announce.GetNext(startkey)
		if err != nil {
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
			if conn == nil {
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
	if err != nil {
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
func sendTooManyRequests(w http.ResponseWriter) {
	sendError(w, "Too Many Requests", http.StatusTooManyRequests)
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
	if err != nil {
		// manually composed error just in case JSON fails
		http.Error(w, `{"code":500,"error":"Internal server Error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	w.Write(text)
}
