// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package payment

import (
	"sync"
	"time"

	"github.com/bitmark-inc/logger"
)

const (
	blockchainCheckInterval = 60 * time.Second
)

// checker periodically extracts possible txs in the latest block
type checker struct {
	log *logger.L
}

func (c *checker) Run(args interface{}, shutdown <-chan struct{}) {
	log := logger.New("checker")
	c.log = log

	log.Info("starting…")
loop:
	for {
		log.Info("begin…")
		select {
		case <-shutdown:
			break loop

		case <-time.After(blockchainCheckInterval): // timeout
			log.Info("checking…")
			var wg sync.WaitGroup
			for _, handler := range globalData.handlers {
				wg.Add(1)
				go handler.checkLatestBlock(&wg)
			}
			log.Debug("waiting…")
			wg.Wait()
		}
	}
}
