// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package block_test

import (
	"fmt"
	"strings"
)

// set this true to get logging ouptuts from various tests
//const verboseTesting = true
const verboseTesting = false

// for dumping the expected hex
func formatBytes(name string, data []byte) string {
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
