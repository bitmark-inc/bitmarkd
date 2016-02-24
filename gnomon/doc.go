// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// a specialised timestamp to index data collections
//
// consists of:
//   seconds (int64)    -> the UTC unix time
//   nano seconds (int) -> fractional time [0 .. 1,000,000,000]
//
// Note that the nano seconds part can be equivalent to 1 second in
// certain cases to allow for a value that is greater than the current
// item, but less than or equal to the current item even if the nano
// seconds was exactly 999,999,999 without having to propagate the
// carry value.
package gnomon
