// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// Detect payment records in the Bitcoin Block Chain
//
// Payments are indicated by OP_RETURN embedded data, this is
// compressed to fit it within the 80 byte Bitcoin limit.
//
// The OP_RETURN(0x6a) data structure:
//
//    +---------------------------------------------------------------------------------+
//    |         1         2         3         4          5         6         7         8|
//    |123456789012345678901234567890123456789012345678 90123456789012345678901234567890|
//    +------------------------------------------------+--------------------------------+
//    |                                                |                                |
//    |     pay id                                     |                                |
//    |                                                |                                |
//    +------------------------------------------------+--------------------------------+
//    |             1         2         3         4    |         1         2         3  |
//    |123456789012345678901234567890123456789012345678|12345678901234567890123456789012|
//    +------------------------------------------------+--------------------------------+
package bitcoin
