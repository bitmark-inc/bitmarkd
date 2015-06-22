// Copyright (c) 2014-2015 Bitmark Inc.
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
func Initialise() {
	if nil != log {
		panic("do not initialise fault twice")
	}
	log = logger.New("PANIC")
	if nil == log {
		panic("failed to get a logger channel")
	}
}

// flush any data
func Finalise() {
	log.Flush()
}

// Log a simple string
func Critical(message string) {

	if _, file, line, ok := runtime.Caller(1); ok {
		log.Criticalf("(%q:%d) "+message, file, line)
	} else {
		log.Critical(message)
	}
}

// Log a formatted string with arguments like fmt.Sprintf()
func Criticalf(format string, arguments ...interface{}) {
	if _, file, line, ok := runtime.Caller(1); ok {
		a := make([]interface{}, 2, 2+len(arguments))
		a[0] = file
		a[1] = line
		a = append(a, arguments...)
		log.Criticalf("(%q:%d) "+format, a...)
	} else {
		log.Criticalf(format, arguments...)
	}
}

// Panic with a formatted message a formatted string with arguments like fmt.Sprintf()
func Panicf(format string, arguments ...interface{}) {
	if _, file, line, ok := runtime.Caller(1); ok {
		a := make([]interface{}, 2, 2+len(arguments))
		a[0] = file
		a[1] = line
		a = append(a, arguments...)
		log.Criticalf("(%q:%d) "+format, a...)
	} else {
		log.Criticalf(format, arguments...)
	}
	Panic("abort, see last messages in log file")
}

// final panic
func Panic(message string) {
	if nil != log {
		log.Criticalf("%s", message)
		log.Flush()                        // make sure log file is saved
		time.Sleep(100 * time.Millisecond) // to allow logging output
	}
	panic(message)
}

// final panic
func PanicWithError(message string, err error) {
	s := fmt.Sprintf("%s failed with error: %v", message, err)
	if nil != log {
		log.Critical(s)
		log.Flush()                        // make sure log file is saved
		time.Sleep(100 * time.Millisecond) // to allow logging output
	}
	panic(s)
}

// conditional panic
func PanicIfError(message string, err error) {
	if nil == err {
		return
	}
	PanicWithError(message, err)
}
