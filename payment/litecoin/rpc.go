// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package litecoin

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

func rpc(url string, reply interface{}) error {

	request, err := http.NewRequest("GET", url, nil)
	if nil != err {
		return err
	}

	response, err := globalData.client.Do(request)
	if nil != err {
		return err
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if nil != err {
		return err
	}
	if http.StatusOK != response.StatusCode {
		return fmt.Errorf("status: %d %q on: %q", response.StatusCode, response.Status, url)
	}
	return json.Unmarshal(body, &reply)
}
