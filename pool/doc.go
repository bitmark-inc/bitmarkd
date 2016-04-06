// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// maintain severa pools of a number of elements in key->value form
//
// This maintains a LevelDB database split into a series of tables.
// Each table is defined by a prefix byte.
package pool
