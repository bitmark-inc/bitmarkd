// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package mine

import (
	"encoding/json"
	"errors"
	"io"
)

// RPC error codes
var (
	ErrOtherUnknown       = errors.New("20 - Other/Unknown")
	ErrJobNotFound        = errors.New("21 - Job not found (=stale)")
	ErrDuplicateShare     = errors.New("22 - Duplicate share")
	ErrLowDifficultyShare = errors.New("23 - Low difficulty share")
	ErrUnauthorizedWorker = errors.New("24 - Unauthorized worker")
	ErrNotSubscribed      = errors.New("25 - Not subscribed")
)

// type to hold error: [code, "description", info]
type rpcError []interface{}

// incoming RPC request
type rpcRequest struct {
	ID     *string       `json:"id"`
	Method string        `json:"method"`
	Params []interface{} `json:"params"`
}

// outgoing successful reply only if "Error" is nil
type rpcResponse struct {
	ID     string      `json:"id"`
	Result interface{} `json:"Result"`
	Error  rpcError    `json:"error"`
}

// success response
func reply(conn io.Writer, id string, r interface{}) error {
	return internalReply(conn, id, r, nil)
}

// failure response
func errorReply(conn io.Writer, id string, e rpcError) error {
	return internalReply(conn, id, nil, e)
}

type Notifier struct {
	io.Writer
}

type NotificationFunc func(conn Notifier, stop <-chan bool, argument interface{})

func ServeConnection(conn io.ReadWriter, server *Server, background NotificationFunc, argument interface{}) error {

	decoder := json.NewDecoder(conn)

	// start the background
	shutdownNotifier := make(chan bool)
	defer close(shutdownNotifier)
	go background(Notifier{conn}, shutdownNotifier, argument)

	for {
		request := rpcRequest{}
		if err := decoder.Decode(&request); err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		go rpcCall(conn, server, *request.ID, request.Method, request.Params)
	}
	return nil
}

// send a notification
func (conn *Notifier) Notify(method string, params []interface{}) error {

	request := rpcRequest{
		ID:     nil,
		Method: method,
		Params: params,
	}
	s, err := json.Marshal(request)
	if err != nil {
		return err
	}

	s = append(s, '\n')
	l := len(s)

	for l > 0 {
		n, err := conn.Write(s)
		if err != nil {
			return err
		}
		l -= n
	}
	return nil
}

// handle the call
func rpcCall(conn io.ReadWriter, server *Server, id string, method string, params []interface{}) error {

	var responseBuffer interface{}

	err := server.Call(method, params, &responseBuffer)
	if err != nil {
		return errorReply(conn, id, rpcMapError(err))
	}

	return reply(conn, id, responseBuffer)
}

// the reply function sends response as '\n' terminated JSON string
func internalReply(conn io.Writer, id string, message interface{}, e rpcError) error {
	r := rpcResponse{
		ID:     id,
		Result: message,
		Error:  e,
	}

	s, err := json.Marshal(r)
	if err != nil {
		return err
	}

	s = append(s, '\n')
	l := len(s)

	for l > 0 {
		n, err := conn.Write(s)
		if err != nil {
			return err
		}
		l -= n
	}
	return nil
}

// change normal error to rpc specific error
func rpcMapError(err error) rpcError {
	switch err {
	default:
		fallthrough
	case ErrOtherUnknown:
		return rpcError{20, "Other/Unknown", nil}
	case ErrJobNotFound:
		return rpcError{21, "Job not found (=stale)", nil}
	case ErrDuplicateShare:
		return rpcError{22, "Duplicate share", nil}
	case ErrLowDifficultyShare:
		return rpcError{23, "Low difficulty share", nil}
	case ErrUnauthorizedWorker:
		return rpcError{24, "Unauthorized worker", nil}
	case ErrNotSubscribed:
		return rpcError{25, "Not subscribed", nil}

	}
}
