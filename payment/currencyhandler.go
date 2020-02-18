// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package payment

import (
	"sync"
)

type currencyHandler interface {
	processPastTxs(dat []byte)
	processIncomingTx(dat []byte)
	checkLatestBlock(wg *sync.WaitGroup)
}
