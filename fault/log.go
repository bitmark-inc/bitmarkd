// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package fault

import (
	"fmt"
	"github.com/bitmark-inc/logger"
	"runtime"
	"time"
)

// hold a logger channel
var log *logger.L

// setup a log channel for last attempt to log something
func Initialise() error {
	if nil != log {
		return ErrAlreadyInitialised
	}
	log = logger.New("PANIC")
	if nil == log {
		return ErrInvalidLoggerChannel
	}
	return nil
}

// flush any data
func Finalise() {
	if nil != log {
		log.Flush()
	}
}

// Log a simple string
func Critical(message string) {
	if _, file, line, ok := runtime.Caller(1); ok {
		internalCriticalf("(%q:%d) "+message, file, line)
	} else {
		internalCriticalf("%s", message)
	}
}

// Log a formatted string with arguments like fmt.Sprintf()
func Criticalf(format string, arguments ...interface{}) {
	if _, file, line, ok := runtime.Caller(1); ok {
		a := make([]interface{}, 2, 2+len(arguments))
		a[0] = file
		a[1] = line
		a = append(a, arguments...)
		internalCriticalf("(%q:%d) "+format, a...)
	} else {
		internalCriticalf(format, arguments...)
	}
}

// Panic with a formatted message a formatted string with arguments like fmt.Sprintf()
func Panicf(format string, arguments ...interface{}) {
	if _, file, line, ok := runtime.Caller(1); ok {
		a := make([]interface{}, 2, 2+len(arguments))
		a[0] = file
		a[1] = line
		a = append(a, arguments...)
		internalCriticalf("(%q:%d) "+format, a...)
	} else {
		internalCriticalf(format, arguments...)
	}
	Panic("abort, see last messages in log file")
}

// final panic
func Panic(message string) {
	internalCriticalf("%s", message)
	time.Sleep(100 * time.Millisecond) // to allow logging output
	panic(message)
}

// final panic
func PanicWithError(message string, err error) {
	s := fmt.Sprintf("%s failed with error: %v", message, err)
	internalCriticalf("%s", s)
	time.Sleep(100 * time.Millisecond) // to allow logging output
	panic(s)
}

// conditional panic
func PanicIfError(message string, err error) {
	if nil == err {
		return
	}
	PanicWithError(message, err)
}

// internal routines to handle uninitilaise logger channel
func internalCriticalf(format string, arguments ...interface{}) {
	if nil == log {
		fmt.Printf("*** "+format+"\n", arguments...)
	} else {
		log.Criticalf(format, arguments...)
		log.Flush() // make sure log file is saved
	}
}
