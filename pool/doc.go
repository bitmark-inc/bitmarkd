// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// maintain a memory pool of a number of elements
//
// This maintains an in memory LRU list upto a fixed number of elements
// and permanently store the data in LevelDB.
package pool
