// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package domain_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/bitmark-inc/bitmarkd/announce/domain"

	"github.com/bitmark-inc/bitmarkd/announce/fixtures"

	"github.com/bitmark-inc/bitmarkd/announce/mocks"
	"github.com/bitmark-inc/logger"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestNewDomain(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	r := mocks.NewMockReceptor(ctl)
	f := func(s string) ([]string, error) { return []string{}, nil }

	_, err := domain.New(logger.New(fixtures.LogCategory), "domain.not.exist", r, f)
	assert.Nil(t, err, "wrong NewDomain")
}

func TestNewDomainWhenLookupError(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	r := mocks.NewMockReceptor(ctl)
	f := func(s string) ([]string, error) {
		return []string{}, fmt.Errorf("error")
	}

	_, err := domain.New(logger.New(fixtures.LogCategory), "domain.not.exist", r, f)
	assert.Equal(t, fmt.Errorf("error"), err, "wrong NewDomain")
}

func TestRunWhenShutdown(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	r := mocks.NewMockReceptor(ctl)
	f := func(s string) ([]string, error) { return []string{}, nil }

	b, _ := domain.New(logger.New(fixtures.LogCategory), "domain.not.exist", r, f)

	shutdown := make(chan struct{})
	wg := new(sync.WaitGroup)
	wg.Add(1)

	go func(wg *sync.WaitGroup) {
		b.Run(nil, shutdown)
		wg.Done()
	}(wg)

	shutdown <- struct{}{}
	wg.Wait()
}
