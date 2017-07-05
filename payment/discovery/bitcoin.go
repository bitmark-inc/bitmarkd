package discovery

import (
	"encoding/json"

	"github.com/bitmark-inc/bitmarkd/payment/bitcoin"
	"github.com/bitmark-inc/logger"
)

type bitcoinHandler struct {
	log *logger.L
}

func (h *bitcoinHandler) recover(dat []byte) {
	txs := make([]bitcoin.Transaction, 0)
	if err := json.Unmarshal(dat, &txs); err != nil {
		h.log.Errorf("unable to unmarshal txs: %v", err)
		return
	}

	for _, tx := range txs {
		h.log.Debugf("retrieved tx (txid = %s) through recovery\n", tx.TxID)
		bitcoin.ScanTx(h.log, &tx)
	}
}

func (h *bitcoinHandler) processTx(data []byte) {
	var tx bitcoin.Transaction
	if err := json.Unmarshal(data, &tx); err != nil {
		h.log.Errorf("unable to unmarshal tx: %v", err)
		return
	}

	h.log.Debugf("retrieved tx (txid = %s) through subscription\n", tx.TxID)
	bitcoin.ScanTx(h.log, &tx)
}
