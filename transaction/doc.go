// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// transaction record handling
//
// contains types for each of the possible transaction records and
// methods to Pack and Unpack them for storage/transfer.
//
// the transactions form a tree with the following limits
// 1. registration record (R) is a root and only one in a tree
// 2. registration transfer records (T) for a single chain trunk
// 3. bitmark transfer records (B) form single chain branches out from the bitmark transfer
// 4. there is only one trunk
// 5. any (T) can have multiple branches
// 6. new branches can only be added at the apex (T)
// 7. branches can only grow at their leaf (B)
//
//  (R)---(T1)---(T2)---(T3)  <---{ Apex (T3) }
//        |  |    |     |  |
//        B  B    B     B  B
//        |  |    |     |  |
//        B  B    B     B  B  <---{ Leaf nodes }
//
//  (R)---(T1)---(T2)---(T3)
//        |  |    |     |  |\
//        B  B    B     B  B B
//        |  |    |     |  |  \
//        B  B    B     B  B   B  <---{ New branch added at Apex (T3) }
//        |             |
//        B             B  <---{ New bitmark transfers at any Leaf }
//
//  (R)---(T1)---(T2)---(T3)---(T4)  <---{ New Apex (T4) }
//        |  |    |     |  |\     |
//        B  B    B     B  B B    B  <---{ New branches only allowed from current Apex }
//        |  |    |     |  |  \
//        B  B    B     B  B   B
//        |             |
//        B             B
package transaction
