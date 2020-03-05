// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package handler

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/rpc"
	"net/rpc/jsonrpc"
	"strconv"
	"strings"
	"time"

	"github.com/bitmark-inc/bitmarkd/rpc/node"

	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/blockheader"
	"github.com/bitmark-inc/bitmarkd/counter"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/p2p"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/util"
	peerlib "github.com/libp2p/go-libp2p-core/peer"

	//	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/bitmarkd/consensus"
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
	if nil != err {
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

	type theReply struct {
		Chain               string        `json:"chain"`
		Mode                string        `json:"mode"`
		Block               blockInfo     `json:"block"`
		RPCs                uint64        `json:"rpcs"`
		Peers               uint64        `json:"peers"`
		TransactionCounters node.Counters `json:"transactionCounters"`
		Difficulty          float64       `json:"difficulty"`
		Hashrate            float64       `json:"hashrate,omitempty"`
		Version             string        `json:"version"`
		Uptime              string        `json:"uptime"`
		PeerID              peerlib.ID    `json:"peerid"`
	}

	reply := theReply{
		Chain: mode.ChainName(),
		Mode:  mode.String(),
		Block: blockInfo{
			LRCount: lrCount{
				Local:  blockheader.Height(),
				Remote: consensus.BlockHeight(),
			},
			Hash: block.LastBlockHash(storage.Pool.Blocks),
		},
		RPCs: connectionCountHTTPS.Uint64(),
		// Miners : mine.ConnectionCount(),
		Difficulty: difficulty.Current.Value(),
		Hashrate:   difficulty.Hashrate(),
		Version:    h.version,
		Uptime:     time.Since(h.start).String(),
		PeerID:     p2p.ID(),
	}

	reply.Peers = uint64(p2p.GetNetworkMetricConnCount())
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
		//ConnectedTo []peerlib.AddrInfo `json:"connectedTo"`
		ConnectedTo []*p2p.Connected `json:"connectedTo"`
	}

	var info reply
	//TODO: Make libp2p version
	info.ConnectedTo = p2p.GetAllPeers()
	sendReply(w, info)
}

// to output peer data
type entry struct {
	PeerID    peerlib.ID `json:"peerid"`
	Listeners []string   `json:"listeners"`
	Timestamp time.Time  `json:"timestamp"`
}

// GET to find data on all peers seen in the announcer
// (restricted to local_allow)
//
// query parameters:
//   public_key=<64-hex-characters>   [32 byte public key in hex]
//   count=<int>                      [1..100  default: 10]
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
	// TODO:  check enCoding part of code
	startkey := r.Form.Get("peerid")

	// count parsing
	count := defaultCount
	n, err := strconv.Atoi(r.Form.Get("count"))
	if nil == err && n >= 1 && n <= maximumCount {
		count = n
	}
	peers := make([]entry, 0, count)

item_loop:
	for i := 0; i < count; i += 1 {
		startID, err := peerlib.IDB58Decode(startkey)
		if err != nil {
			h.log.Warnf("ID DecodeBase58 Error :%v ID::%v", err, startkey)
			continue item_loop
		}
		id, listeners, timestamp, err := announce.GetNext(startID)
		if nil != err {
			sendInternalServerError(w)
			return
		}
		// p2p version
		if util.IDCompare(id, startID) <= 0 {
			break item_loop
		}
		startID = id
		p := id
		stringAddrs := util.MaAddrToString(listeners)
		peers = append(peers, entry{
			PeerID:    p,
			Listeners: stringAddrs,
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
	if nil != err {
		// manually composed error just in case JSON fails
		http.Error(w, `{"code":500,"error":"Internal server Error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	w.Write(text)
}
