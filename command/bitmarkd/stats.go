// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"runtime"
	"time"

	"github.com/bitmark-inc/logger"
)

const (
	statsDelay = 60 * time.Second
	mega       = 1048576
)

func memstats() {

	log := logger.New("memory")

	for {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		text, err := json.Marshal(m)
		if nil != err {
			log.Errorf("marshal error: %s", err)
		} else {
			log.Infof("stats: %s", text)
		}
		a := m.Alloc / mega
		t := m.TotalAlloc / mega
		s := m.Sys / mega
		log.Warnf("allocated: %d M  cumulative: %d M  OS virtual: %d M", a, t, s)

		time.Sleep(statsDelay)
	}
}
