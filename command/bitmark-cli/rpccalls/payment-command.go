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

	c := strings.ToLower(currency.String())

	command := ""
	if testnet {
		command = fmt.Sprintf(paymentCommandTest, c, payId)
	} else {
		command = fmt.Sprintf(paymentCommandLive, c, payId)
	}

	for _, p := range payments {
		command += fmt.Sprintf(" '%s,%d'", p.Address, p.Amount)
	}
	return command
}
