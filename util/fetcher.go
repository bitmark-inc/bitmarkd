// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package util

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// FetchJSON - fetch a JSON response from an HTTP request and decode
// it
func FetchJSON(client *http.Client, url string, reply interface{}) error {
	request, err := http.NewRequest("GET", url, http.NoBody)
	if err != nil {
		return err
	}

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	if http.StatusOK != response.StatusCode {
		return fmt.Errorf("status: %d %q on: %q", response.StatusCode, response.Status, url)
	}
	return json.Unmarshal(body, &reply)
}
