// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package filters

import (
	"fmt"
	"sync"

	"github.com/bitmark-inc/logger"
)

// Camm - filter type
type Camm struct {
	sync.RWMutex
	Filter
	nMedian uint64
	nWMA    uint64
	f       []Filter
	current float64
}

// NewCamm - create a filter instance
func NewCamm(start float64, nMedian uint64, nWMA uint64) Filter {
	filter := Camm{
		nMedian: nMedian,
		nWMA:    nWMA,
	}
	filter.f = make([]Filter, 2)
	filter.f[0] = NewSMM(start, nMedian)
	filter.f[1] = NewWMA(start, nWMA)

	return &filter
}

// Name - return the name of the filter
func (filter *Camm) Name() string {
	filter.RLock()
	defer filter.RUnlock()

	return fmt.Sprintf("Camm %d,%d", filter.nMedian, filter.nWMA)
}

// Process - add a input value to the filter
func (filter *Camm) Process(s float64) float64 {
	filter.Lock()
	defer filter.Unlock()
	if s < 0 {
		logger.Panicf("camm negative sample: %f", s)
	}

	for _, f := range filter.f {
		s = f.Process(s)
	}
	filter.current = s
	return filter.current
}

// Current - return the current filter value
func (filter *Camm) Current() float64 {
	filter.RLock()
	defer filter.RUnlock()

	return filter.current
}
