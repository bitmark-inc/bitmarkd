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

// WMA - filter instance
type WMA struct {
	sync.RWMutex
	Filter
	samples     []float64
	it          int
	current     float64
	n           float64
	total       float64
	numerator   float64
	denominator float64
}

// NewWMA - create a filter instance
func NewWMA(start float64, n uint64) Filter {
	filter := WMA{
		samples:     make([]float64, n),
		current:     start,
		n:           float64(n),
		total:       float64(n), // * (n + 1) / 2),
		numerator:   float64(n * (n + 1) / 2),
		denominator: float64(n * (n + 1) / 2),
	}
	for i := uint64(0); i < n; i += 1 {
		filter.samples[i] = start
	}
	return &filter
}

// Name - return the name of the filter
func (filter *WMA) Name() string {
	filter.RLock()
	defer filter.RUnlock()

	return fmt.Sprintf("Weighted Moving Average %d", len(filter.samples))
}

// Process - add a input value to the filter
func (filter *WMA) Process(s float64) float64 {
	filter.Lock()
	defer filter.Unlock()

	if s < 0 {
		logger.Panicf("wma negative sample: %f", s)
	}

	filter.numerator += filter.n*s - filter.total

	filter.total += s - filter.samples[filter.it]

	filter.samples[filter.it] = s

	if filter.it += 1; filter.it >= len(filter.samples) {
		filter.it = 0
	}

	filter.current = filter.numerator / filter.denominator
	if filter.current < 0 {
		filter.current = 0
	}

	return filter.current
}

// Current - return the current filter value
func (filter *WMA) Current() float64 {
	filter.RLock()
	defer filter.RUnlock()

	return filter.current
}
