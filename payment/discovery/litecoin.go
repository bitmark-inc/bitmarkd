package discovery

import (
	"encoding/json"

	"github.com/bitmark-inc/bitmarkd/payment/litecoin"
	"github.com/bitmark-inc/logger"
)

type litecoinHandler struct {
	log *logger.L
}

func (h *litecoinHandler) recover(dat []byte) {
	txs := make([]litecoin.Transaction, 0)
	if err := json.Unmarshal(dat, &txs); err != nil {
		h.log.Errorf("unable to unmarshal txs: %v", err)
		return
	}

	for _, tx := range txs {
		h.log.Debugf("retrieved tx (txid = %s) through recovery\n", tx.TxId)
		litecoin.CheckForPaymentTransaction(h.log, &tx)
	}
}

func (h *litecoinHandler) processTx(data []byte) {
	var tx litecoin.Transaction
	if err := json.Unmarshal(data, &tx); err != nil {
		h.log.Errorf("unable to unmarshal tx: %v", err)
		return
	}

	h.log.Debugf("retrieved tx (txid = %s) through subscription\n", tx.TxId)
	litecoin.CheckForPaymentTransaction(h.log, &tx)
}
