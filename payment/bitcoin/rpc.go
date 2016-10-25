// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package bitcoin

import (
	"bytes"
	"encoding/json"
	"github.com/bitmark-inc/bitmarkd/fault"
	"io/ioutil"
	"net/http"
)

// high level call - only use while global data locked
// because the HTTP RPC cannot interleave calls and responses
func bitcoinCall(method string, params []interface{}, reply interface{}) error {
	if !globalData.initialised {
		fault.Panic("bitcoin not initialised")
	}

	globalData.id += 1

	arguments := bitcoinArguments{
		Id:     globalData.id,
		Method: method,
		Params: params,
	}
	response := bitcoinReply{
		Result: reply,
	}
	globalData.log.Debugf("rpc call with: %v", arguments)
	err := bitcoinRPC(&arguments, &response)
	if nil != err {
		globalData.log.Tracef("rpc returned error: %v", err)
		return err
	}

	if nil != response.Error {
		s := response.Error.Message
		return fault.ProcessError("Bitcoin RPC error: " + s)
	}
	return nil
}

// for encoding the RPC arguments
type bitcoinArguments struct {
	Id     uint64        `json:"id"`
	Method string        `json:"method"`
	Params []interface{} `json:"params"`
}

// the RPC error response
type bitcoinRpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// for decoding the RPC reply
type bitcoinReply struct {
	Id     int64            `json:"id"`
	Method string           `json:"method"`
	Result interface{}      `json:"result"`
	Error  *bitcoinRpcError `json:"error"`
}

// basic RPC - only use while global data locked
func bitcoinRPC(arguments *bitcoinArguments, reply *bitcoinReply) error {

	s, err := json.Marshal(arguments)
	if nil != err {
		return err
	}

	globalData.log.Tracef("rpc send: %s", s)

	postData := bytes.NewBuffer(s)

	request, err := http.NewRequest("POST", globalData.url, postData)
	if nil != err {
		return err
	}
	request.SetBasicAuth(globalData.username, globalData.password)

	response, err := globalData.client.Do(request)
	if nil != err {
		return err
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if nil != err {
		return err
	}

	globalData.log.Tracef("rpc response body: %s", body)

	err = json.Unmarshal(body, &reply)
	if nil != err {
		return err
	}

	globalData.log.Debugf("rpc receive: %s", body)

	return nil
}
