// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// Generate a large rate of issues for testing
//
// e.g. to generate issues at 5.0 persecond for five minutes:
//      (add -v flag to sse JSON requests and responses)
//
//   issue-generator [-v] rate 5.0 5
package main
