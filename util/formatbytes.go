// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package util

import (
	"fmt"
	"strings"
)

// FormatBytes - for dumping the expected hex used by some test
// routines
func FormatBytes(name string, data []byte) string {
	a := strings.Split(fmt.Sprintf("% #x", data), " ")
	s := name + " := []byte{"
	n := 8
	for i := 0; i < len(a); i += 1 {
		n += 1
		if n >= 8 {
			s += "\n\t"
			n = 0
		}
		s += a[i] + ", "
	}
	return s + "\n}"
}
