// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"os/exec"
)

const (
	passwordTag = "bitmark-cli:password:"
)

// expect to execute agent with parameters
//   --confirm=1         - for additional confirm
//   cache-id            - alows password to be cached for a time
//   error-message       - blank
//   prompt              - names the identity
//   description         - shows creat/transfer opration
func passwordFromAgent(name string, title string, agent string, clear bool) (string, error) {

	cacheId := passwordTag + name
	errorMessage := ""
	prompt := "Password for: " + name
	description := "Enter password to: " + title

	arguments := []string{}
	if clear {
		arguments = append(arguments, "--clear")
	}
	arguments = append(arguments, []string{
		"--confirm=1",
		cacheId,
		errorMessage,
		prompt,
		description}...)

	out, err := exec.Command(agent, arguments...).Output()
	return string(out), err
}
