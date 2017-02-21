// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package filters

import (
	"fmt"
	"sort"
	"sync"
)

type SMM struct {
	sync.RWMutex
	Filter
	samples []float64
	it      int
	points  float64
	current float64
}

func NewSMM(start float64, n uint64) Filter {
	if 0 == n%2 {
		panic("need odd number of samples")
	}

	filter := SMM{
		samples: make([]float64, n),
		points:  float64(n),
		current: start,
	}
	for i := range filter.samples {
		filter.samples[i] = start
	}
	return &filter
}

func (filter *SMM) Name() string {
	filter.RLock()
	defer filter.RUnlock()

	return fmt.Sprintf("Simple Moving Median %d", len(filter.samples))
}

func (filter *SMM) Process(s float64) float64 {
	filter.Lock()
	defer filter.Unlock()

	filter.samples[filter.it] = s

	if filter.it += 1; filter.it >= len(filter.samples) {
		filter.it = 0
	}

	a := make([]float64, len(filter.samples))
	copy(a, filter.samples)
	sort.Float64s(a)

	filter.current = a[len(filter.samples)/2+1]
	return filter.current
}

func (filter *SMM) Current() float64 {
	filter.RLock()
	defer filter.RUnlock()

	return filter.current
}
