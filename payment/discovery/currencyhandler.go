package discovery

type currencyHandler interface {
	recover(dat []byte)   // process missed txs
	processTx(dat []byte) // process new tx
}
