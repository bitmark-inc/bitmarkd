// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/encrypt"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/keypair"
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

func promptPasswordReader() (string, error) {
	console, fd, state := getTerminal()
	password, err := console.ReadPassword("Set identity password(length >= 8): ")
	if nil != err {
		fmt.Printf("Get password fail: %s\n", err)
		return "", err
	}
	terminal.Restore(fd, state)

	passwordLen := len(password)
	if passwordLen < 8 {
		return "", fault.ErrInvalidPasswordLength
	}

	console, fd, state = getTerminal()
	verifyPassword, err := console.ReadPassword("Verify password: ")
	if nil != err {
		fmt.Printf("verify failed: %s\n", err)
		return "", fault.ErrPasswordMismatch
	}
	terminal.Restore(fd, state)

	if password != verifyPassword {
		return "", fault.ErrPasswordMismatch
	}

	return password, nil
}

func promptCheckPasswordReader() (string, error) {
	console, fd, state := getTerminal()
	password, err := console.ReadPassword("password: ")
	if nil != err {
		fmt.Printf("Get password fail: %s\n", err)
		return "", err
	}
	terminal.Restore(fd, state)

	return password, nil
}

func promptAndCheckPassword(issuer *encrypt.IdentityType) (*keypair.KeyPair, error) {
	password, err := promptCheckPasswordReader()
	if nil != err {
		return nil, err
	}

	keyPair, err := encrypt.VerifyPassword(password, issuer)
	if nil != err {
		return nil, err
	}
	return keyPair, nil
}
