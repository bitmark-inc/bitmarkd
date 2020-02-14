// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package domain_test

import (
	"sync"
	"testing"

	"github.com/bitmark-inc/logger"

	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/bitmarkd/announce/mocks"

	"github.com/golang/mock/gomock"

	"github.com/bitmark-inc/bitmarkd/announce/domain"
)

func TestNewDomain(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	r := mocks.NewMockReceptor(ctl)
	f := func(s string) ([]string, error) { return []string{}, nil }

	_, err := domain.NewDomain(logger.New(logCategory), "domain.not.exist", r, f)
	assert.Nil(t, err, "wrong NewDomain")
}

func TestRunWhenShutdown(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	r := mocks.NewMockReceptor(ctl)
	f := func(s string) ([]string, error) { return []string{}, nil }

	b, _ := domain.NewDomain(logger.New(logCategory), "domain.not.exist", r, f)

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
