// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/logger"
	"io"
	"net/http"
	"net/rpc"
	"net/rpc/jsonrpc"
	"strconv"
	"strings"
	"time"
)

// defaults
const (
	defaultCount = 10
	maximumCount = 100
)

// type to allow rpc system to interface to http request
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
	Log        *logger.L
	Version    string
	Server     *rpc.Server
	Node       *Node
	LocalAllow map[string]struct{}
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

	server := s.Server

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

// to allow a GET for the same response and Node.Info RPC
func (s *httpHandler) info(w http.ResponseWriter, r *http.Request) {
	if http.MethodGet != r.Method {
		sendMethodNotAllowed(w)
		return
	}
	connectionCount.Increment()
	defer connectionCount.Decrement()

	var info InfoReply
	err := s.Node.Info(nil, &info)
	if nil != err {
		sendInternalServerError(w)
		return
	}

	sendReply(w, info)
}

func (s *httpHandler) connectors(w http.ResponseWriter, r *http.Request) {
	if http.MethodGet != r.Method {
		sendMethodNotAllowed(w)
		return
	}
	connectionCount.Increment()
	defer connectionCount.Decrement()

	var info ConnectorReply
	err := s.Node.Connectors(nil, &info)
	if nil != err {
		sendInternalServerError(w)
		return
	}

	sendReply(w, info)
}

func (s *httpHandler) subscribers(w http.ResponseWriter, r *http.Request) {
	if http.MethodGet != r.Method {
		sendMethodNotAllowed(w)
		return
	}
	connectionCount.Increment()
	defer connectionCount.Decrement()

	var info SubscriberReply
	err := s.Node.Subscribers(nil, &info)
	if nil != err {
		sendInternalServerError(w)
		return
	}

	sendReply(w, info)
}

// to output peer data
type entry struct {
	PublicKey  string    `json:"public_key"`
	Broadcasts []string  `json:"broadcasts"`
	Listeners  []string  `json:"listeners"`
	Timestamp  time.Time `json:"timestamp"`
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

	last := strings.LastIndex(r.RemoteAddr, ":")
	if last >= 0 {
		addr := r.RemoteAddr[:last]
		if _, ok := s.LocalAllow[addr]; ok {
			goto allow_access
		}
	}
	s.Log.Warnf("Deny access: %q", r.RemoteAddr)
	sendForbidden(w)
	return // *IMPORTANT*

allow_access:

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
		publicKey, broadcasts, listeners, timestamp, err := announce.GetNext(startkey)
		if nil != err {
			sendInternalServerError(w)
			return
		}
		if bytes.Compare(publicKey, startkey) <= 0 {
			break item_loop
		}
		startkey = publicKey

		p := hex.EncodeToString(publicKey)

		bPack := util.PackedConnection(broadcasts)
		bc := make([]string, 0, 2)
	bc_loop:
		for {
			conn, n := bPack.Unpack()
			if nil == conn {
				break bc_loop
			}
			bc = append(bc, conn.String())
			bPack = bPack[n:]
		}
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
			PublicKey:  p,
			Broadcasts: bc,
			Listeners:  lc,
			Timestamp:  timestamp,
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
