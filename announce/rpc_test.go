// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package announce_test

import (
	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/announce/fingerprint"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/logger"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
)

const (
	testingDirName = "testing"
)

func setupTestLogger() {
	removeFiles()
	_ = os.Mkdir(testingDirName, 0700)

	logging := logger.Configuration{
		Directory: testingDirName,
		File:      "testing.log",
		Size:      1048576,
		Count:     10,
		Console:   false,
		Levels: map[string]string{
			logger.DefaultTag: "critical",
		},
	}

	// start logging
	_ = logger.Initialise(logging)
}

func teardownTestLogger() {
	removeFiles()
}

func removeFiles() {
	_ = os.RemoveAll(testingDirName)
}

func TestSetRPC(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()
	defer announce.Finalise()

	f := func(string) ([]string, error) {
		return []string{}, nil
	}
	_ = announce.Initialise("random.test.domain", "", announce.DnsOnly, f)

	fp := fingerprint.Type{1, 2, 3, 4, 5}
	err := announce.SetRPC(fp, []byte{6, 7, 8, 9, 10})
	assert.Nil(t, err, "wrong set rpc error")

	startIndex := uint64(0)
	rpcs, start, err := announce.FetchRPCs(startIndex, 10)
	assert.Nil(t, err, "wrong fetch")
	assert.Equal(t, 1, len(rpcs), "wrong rpc length")
	assert.Equal(t, fp, rpcs[0].Fingerprint, "wrong fingerprint")
	assert.Equal(t, startIndex+1, start, "wrong startIndex")
}

func TestSetRPCWhenRepeat(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()
	defer announce.Finalise()

	f := func(string) ([]string, error) {
		return []string{}, nil
	}
	_ = announce.Initialise("random.test.domain", "", announce.DnsOnly, f)

	fp := fingerprint.Type{1, 2, 3, 4, 5}
	r := []byte{6, 7, 8, 9, 10}
	err := announce.SetRPC(fp, r)
	err = announce.SetRPC(fp, r)
	assert.Equal(t, fault.AlreadyInitialised, err, "wrong second initialise")
}

func TestFetchRPCsWhenCountLessZero(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()
	defer announce.Finalise()

	f := func(string) ([]string, error) {
		return []string{}, nil
	}
	_ = announce.Initialise("random.test.domain", "", announce.DnsOnly, f)

	fp := fingerprint.Type{1, 2, 3, 4, 5}
	err := announce.SetRPC(fp, []byte{6, 7, 8, 9, 10})
	assert.Nil(t, err, "wrong set rpc error")

	_, _, err = announce.FetchRPCs(uint64(0), -1)
	assert.Equal(t, fault.InvalidCount, err, "wrong fetch")
}

func TestFetchRPCsWhenStartTooLarge(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()
	defer announce.Finalise()

	f := func(string) ([]string, error) {
		return []string{}, nil
	}
	_ = announce.Initialise("random.test.domain", "", announce.DnsOnly, f)

	fp := fingerprint.Type{1, 2, 3, 4, 5}
	err := announce.SetRPC(fp, []byte{6, 7, 8, 9, 10})
	assert.Nil(t, err, "wrong set rpc error")

	rpcs, start, err := announce.FetchRPCs(uint64(5), 10)
	assert.Nil(t, err, "wrong fetch")
	assert.Equal(t, 0, len(rpcs), "wrong list")
	assert.Equal(t, uint64(0), start, "wrong start")
}

func TestAddRPC(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()
	defer announce.Finalise()

	f := func(string) ([]string, error) {
		return []string{}, nil
	}
	_ = announce.Initialise("random.test.domain", "", announce.DnsOnly, f)

	fp := make([]byte, 32)
	fp[0] = 1
	r := []byte{6, 7, 8, 9}
	ts := uint64(time.Now().Unix())
	result := announce.AddRPC(fp, r, ts)
	assert.Equal(t, true, result, "wrong add rpc")

	rpcs, _, err := announce.FetchRPCs(uint64(0), 10)
	assert.Nil(t, err, "wrong fetch")
	assert.Equal(t, 1, len(rpcs), "wrong list")
}

func TestAddRPCWhenInvalidFingerprint(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()
	defer announce.Finalise()

	f := func(string) ([]string, error) {
		return []string{}, nil
	}
	_ = announce.Initialise("random.test.domain", "", announce.DnsOnly, f)

	fp := make([]byte, 30)
	fp[0] = 1
	r := []byte{6, 7, 8, 9}
	ts := uint64(time.Now().Unix())
	result := announce.AddRPC(fp, r, ts)
	assert.Equal(t, false, result, "wrong add rpc")
}

func TestAddRPCWhenInvalidRPCs(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()
	defer announce.Finalise()

	f := func(string) ([]string, error) {
		return []string{}, nil
	}
	_ = announce.Initialise("random.test.domain", "", announce.DnsOnly, f)

	fp := make([]byte, 32)
	fp[0] = 1
	r := make([]byte, 200)
	r[0] = 5
	ts := uint64(time.Now().Unix())
	result := announce.AddRPC(fp, r, ts)
	assert.Equal(t, false, result, "wrong add rpc")
}

func TestAddRPCWhenExpired(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()
	defer announce.Finalise()

	f := func(string) ([]string, error) {
		return []string{}, nil
	}
	_ = announce.Initialise("random.test.domain", "", announce.DnsOnly, f)

	fp := make([]byte, 32)
	fp[0] = 1
	r := []byte{6, 7, 8, 9}
	result := announce.AddRPC(fp, r, uint64(0))
	assert.Equal(t, false, result, "wrong add rpc")
}
