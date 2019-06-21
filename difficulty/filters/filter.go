// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package filters

// Filter - interface for filter modules
type Filter interface {
	Process(s float64) float64
	Current() float64
	Name() string
}
