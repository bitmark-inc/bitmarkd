// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"

	"github.com/bitmark-inc/bitmarkd/fault"
	"golang.org/x/crypto/ssh/terminal"
)

var passwordConsole *terminal.Terminal

func getTerminal() (*terminal.Terminal, int, *terminal.State) {
	oldState, err := terminal.MakeRaw(0)
	if err != nil {
		panic(err)
	}

	if passwordConsole != nil {
		return passwordConsole, 0, oldState
	}

	tmpIO, err := os.OpenFile("/dev/tty", os.O_RDWR, os.ModePerm)
	if err != nil {
		panic("No console")
	}

	passwordConsole = terminal.NewTerminal(tmpIO, "bitmark-cli: ")

	return passwordConsole, 0, oldState
}

func promptNewPassword() (string, error) {
	console, fd, state := getTerminal()
	password, err := console.ReadPassword("Set new password (length >= 8): ")
	terminal.Restore(fd, state)

	if err != nil {
		fmt.Printf("Get password fail: %s\n", err)
		return "", err
	}

	passwordLen := len(password)
	if passwordLen < 8 {
		return "", fault.InvalidPasswordLength
	}

	console, fd, state = getTerminal()
	verifyPassword, err := console.ReadPassword("Confirm new password: ")
	terminal.Restore(fd, state)

	if err != nil {
		fmt.Printf("verify failed: %s\n", err)
		return "", fault.PasswordMismatch
	}

	if password != verifyPassword {
		return "", fault.PasswordMismatch
	}

	return password, nil
}

func promptPassword(name string) (string, error) {
	console, fd, state := getTerminal()
	password, err := console.ReadPassword("Password for: " + name + ": ")
	terminal.Restore(fd, state)

	if err != nil {
		fmt.Printf("read password error: %s\n", err)
		return "", err
	}

	return password, nil
}
