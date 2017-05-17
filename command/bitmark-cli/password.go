package main

import (
	"fmt"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/keypair"
	"golang.org/x/crypto/ssh/terminal"
	"os"
)

var (
	ErrPasswordLength   = fault.InvalidError("password length is invalid")
	ErrVerifiedPassword = fault.InvalidError("verified password is different")
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
		return "", ErrPasswordLength
	}

	console, fd, state = getTerminal()
	verifyPassword, err := console.ReadPassword("Verify password: ")
	if nil != err {
		fmt.Printf("verify failed: %s\n", err)
		return "", ErrVerifiedPassword
	}
	terminal.Restore(fd, state)

	if password != verifyPassword {
		return "", ErrVerifiedPassword
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

func promptAndCheckPassword(issuer *keypair.IdentityType) (*keypair.KeyPair, error) {
	password, err := promptCheckPasswordReader()
	if nil != err {
		return nil, err
	}

	keyPair, err := keypair.VerifyPassword(password, issuer)
	if nil != err {
		return nil, err
	}
	return keyPair, nil
}
