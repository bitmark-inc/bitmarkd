// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package reservoir

import (
	"time"

	"github.com/bitmark-inc/logger"
)

const expirationCheckInterval = 5 * time.Minute

type cleaner struct {
	log *logger.L
}

func (c *cleaner) Run(args interface{}, shutdown <-chan struct{}) {

	c.log = logger.New("expiration")

	ticker := time.NewTicker(expirationCheckInterval)
	for {
		select {
		case <-ticker.C:
			c.deleteExpiredItems()
		case <-shutdown:
			ticker.Stop()
			return
		}
	}
}

func (c *cleaner) deleteExpiredItems() {

	globalData.Lock()
	for key, item := range globalData.pendingTransactions {
		if expired(item.expiresAt) {
			internalDelete(key)
		}
	}
	for key, item := range globalData.pendingFreeIssues {
		if expired(item.expiresAt) {
			internalDelete(key)
		}
	}
	for key, item := range globalData.pendingPaidIssues {
		if expired(item.expiresAt) {
			internalDelete(key)
		}
	}
	globalData.Unlock()
}

func expired(exp time.Time) bool {
	return exp.IsZero() || time.Since(exp) > 0
}
