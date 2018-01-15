// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpccalls

import (
	"fmt"
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"strings"
)

// prefix for the payment command
// assumed format is: paymentCommand 'PaymentId' 'address₁,SatoshiAmount₁' … 'addressN,SatoshiAmountN'
const (
	paymentCommandLive = "bitmark-wallet --conf ${XDG_CONFIG_HOME}/bitmark-wallet/bitmark-wallet.conf %s sendmany --hex-data '%s'"
	paymentCommandTest = "bitmark-wallet --conf ${XDG_CONFIG_HOME}/bitmark-wallet/bitmark-wallet.conf %s --testnet sendmany --hex-data '%s'"
)

func paymentCommand(testnet bool, currency currency.Currency, payId string, payments transactionrecord.PaymentAlternative) string {

	f := ""
	if testnet {
		f = paymentCommandTest
	} else {
		f = paymentCommandLive
	}

	c := strings.ToLower(currency.String())
	command := fmt.Sprintf(f, c, payId)

	for _, p := range payments {
		command += fmt.Sprintf(" '%s,%d'", p.Address, p.Amount)
	}
	return command
}
