// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/bitmark-inc/bitmarkd/fault"
)

var passwordConsole *terminal.Terminal

func getTerminal() (*terminal.Terminal, int, *terminal.State) {
	oldState, err := terminal.MakeRaw(0)
	if err != nil {
		panic(err)
	}

	if nil != passwordConsole {
		return passwordConsole, 0, oldState
	}

	tmpIO, err := os.OpenFile("/dev/tty", os.O_RDWR, os.ModePerm)
	if nil != err {
		panic("No console")
	}

	passwordConsole = terminal.NewTerminal(tmpIO, "bitmark-cli: ")

	return passwordConsole, 0, oldState
}

func promptNewPassword() (string, error) {
	console, fd, state := getTerminal()
	password, err := console.ReadPassword("Set new password (length >= 8): ")
	terminal.Restore(fd, state)

	if nil != err {
		fmt.Printf("Get password fail: %s\n", err)
		return "", err
	}

	passwordLen := len(password)
	if passwordLen < 8 {
		return "", fault.ErrInvalidPasswordLength
	}

	console, fd, state = getTerminal()
	verifyPassword, err := console.ReadPassword("Confirm new password: ")
	terminal.Restore(fd, state)

	if nil != err {
		fmt.Printf("verify failed: %s\n", err)
		return "", fault.ErrPasswordMismatch
	}

	if password != verifyPassword {
		return "", fault.ErrPasswordMismatch
	}

	return password, nil
}

func promptPassword(name string) (string, error) {
	console, fd, state := getTerminal()
	password, err := console.ReadPassword("Password for: " + name + ": ")
	terminal.Restore(fd, state)

	if nil != err {
		fmt.Printf("read password error: %s\n", err)
		return "", err
	}

	return password, nil
}
