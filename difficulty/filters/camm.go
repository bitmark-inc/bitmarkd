// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package filters

import (
	"fmt"
	"sync"
)

type Camm struct {
	sync.RWMutex
	Filter
	nMedian uint64
	nWMA    uint64
	f       []Filter
	current float64
}

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

func (filter *Camm) Name() string {
	filter.RLock()
	defer filter.RUnlock()

	return fmt.Sprintf("Camm %d,%d", filter.nMedian, filter.nWMA)
}

func (filter *Camm) Process(s float64) float64 {
	filter.Lock()
	defer filter.Unlock()

	for _, f := range filter.f {
		s = f.Process(s)
	}
	filter.current = s
	return filter.current
}

func (filter *Camm) Current() float64 {
	filter.RLock()
	defer filter.RUnlock()

	return filter.current
}
