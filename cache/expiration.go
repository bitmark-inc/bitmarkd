// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package cache

import (
	"reflect"
	"time"
)

const expirationCheckInterval = 5 * time.Minute

type cleaner struct{}

func (c *cleaner) Run(args interface{}, shutdown <-chan struct{}) {
	ticker := time.NewTicker(expirationCheckInterval)
	for {
		select {
		case <-ticker.C:
			deleteExpiredItems()
		case <-shutdown:
			ticker.Stop()
			return
		}
	}
}

func deleteExpiredItems() {
	poolType := reflect.TypeOf(Pool)
	poolValue := reflect.ValueOf(&Pool).Elem()

	for i := 0; i < poolType.NumField(); i++ {
		poolData := poolValue.Field(i).Interface().(*poolData)

		poolData.Lock()
		for key, item := range poolData.items {
			if expired(item.expiresAt) {
				delete(poolData.items, key)
			}
		}
		poolData.Unlock()
	}
}

func expired(exp time.Time) bool {
	return !exp.IsZero() && time.Since(exp) > 0
}
