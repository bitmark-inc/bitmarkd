// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package payment

import (
	"bytes"
	"net"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/btcsuite/btcd/addrmgr"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/connmgr"
	"github.com/btcsuite/btcd/peer"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/patrickmn/go-cache"

	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/currency/litecoin"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/logger"
)

const checkpointBackLimit = 2000
const MaximumOutboundPeers = 32
const PaymentExpiry = 12 * time.Hour
const HeaderSyncTimeout = time.Minute

// p2pWatcher is a watcher that sync with bitcoin / litecoin blockchain by its peer to peer protocol.
type p2pWatcher struct {
	sync.WaitGroup

	connectedPeers *PeerMap
	currency       currency.Currency

	bootstrapNodes []string
	addrManager    *addrmgr.AddrManager
	connManager    *connmgr.ConnManager
	networkParams  *chaincfg.Params
	srcAddr        *wire.NetAddress
	checkpoint     chaincfg.Checkpoint
	storage        storage.P2PStorage
	blockCache     *cache.Cache
	log            *logger.L

	lastHash     *chainhash.Hash
	lastHeight   int32
	onHeadersErr chan error
	stopping     bool
	shutdown     chan struct{}
}

func newP2pWatcher(c currency.Currency, peerDirectory string, bootstrapNodes []string) (*p2pWatcher, error) {
	var attemptLock sync.Mutex
	log := logger.New(c.String() + "_watcher")
	var paymentStore storage.P2PStorage
	switch c {
	case currency.Bitcoin:
		paymentStore = storage.PaymentStorage.Btc
	case currency.Litecoin:
		paymentStore = storage.PaymentStorage.Ltc
	default:
		return nil, fault.UnsupportedCurrency
	}

	networkParams := c.ChainParam(mode.ChainName())

	defaultPort, err := strconv.ParseInt(networkParams.DefaultPort, 10, 16)
	if err != nil {
		return nil, err
	}
	log.Tracef("watcher default port: %d", defaultPort)

	addrManager := addrmgr.New(peerDirectory, nil)

	w := &p2pWatcher{
		currency:       c,
		connectedPeers: NewPeerMap(),
		bootstrapNodes: bootstrapNodes,
		addrManager:    addrManager,
		networkParams:  networkParams,
		srcAddr:        wire.NewNetAddressIPPort(net.ParseIP("0.0.0.0"), uint16(defaultPort), 0),
		storage:        paymentStore,
		blockCache:     cache.New(time.Hour, 2*time.Hour),
		log:            log,
		onHeadersErr:   make(chan error),
		shutdown:       make(chan struct{}),
	}

	if l := len(networkParams.Checkpoints); l > 0 {
		w.checkpoint = networkParams.Checkpoints[l-1]
	} else {
		w.checkpoint = chaincfg.Checkpoint{
			Hash:   networkParams.GenesisHash,
			Height: 0,
		}
	}

	//	prepare configuration for the connection manager
	config := connmgr.Config{
		TargetOutbound:  MaximumOutboundPeers,
		OnConnection:    w.onConnectionConnected,
		OnDisconnection: w.onConnectionDisconnected,
		GetNewAddress: func() (net.Addr, error) {
			ka := addrManager.GetAddress()
			if ka == nil {
				return nil, fault.NoAddressToReturn
			}
			address := ka.NetAddress()
			addr := &net.TCPAddr{
				Port: int(address.Port),
				IP:   address.IP,
			}
			attemptLock.Lock()
			defer attemptLock.Unlock()

			if time.Since(ka.LastAttempt()) < 10*time.Minute {
				return nil, fault.NoAddressToReturn
			}

			if w.connectedPeers.Exist(addr.String()) {
				w.log.Warnf("ignore connected peer: %s", addr.String())
				return nil, fault.NoAddressToReturn
			}

			addrManager.Attempt(address)
			return addr, nil
		},
		Dial: func(addr net.Addr) (net.Conn, error) {
			return net.Dial("tcp", addr.String())
		},
	}

	connManager, err := connmgr.New(&config)
	if err != nil {
		return nil, err
	}

	w.connManager = connManager

	lastHash, err := w.storage.GetCheckpoint()
	if err != nil {
		log.Warnf("unable to get checkpoint: %s", err)
	}

	if lastHash != nil {
		lastHeight, err := w.storage.GetHeight(lastHash)
		if err != nil {
			w.log.Warnf("unable to get last hash: %s", err)
		} else {
			w.lastHash = lastHash
			w.lastHeight = lastHeight
		}
	}

	// Since lastHeight is zero, we will reset the data from the checkpoint
	if w.lastHeight == 0 {
		w.lastHash = w.checkpoint.Hash
		w.lastHeight = w.checkpoint.Height

		// Write the first hash data into storage
		if err := w.storage.StoreBlock(w.lastHeight, w.lastHash); err != nil {
			return nil, err
		}
	}
	return w, nil
}

// syncHeader will submit a `GetHeaders command to bitcoin peer and wait for its
// response to be processed
func (w *p2pWatcher) syncHeaderFromPeer(p *peer.Peer) error {
	hash := w.lastHash

	if p.LastBlock() < w.lastHeight {
		var err error
		hash, err = w.storage.GetHash(p.LastBlock())

		if err != nil {
			return err
		}
	}

	w.log.Infof("Fetch headers from last block hash: %s", hash)
	headerMsg := wire.NewMsgGetHeaders()
	headerMsg.AddBlockLocatorHash(hash)
	p.QueueMessage(headerMsg, nil)

	select {
	case <-w.shutdown:
		return fault.ProcessStopping
	case err := <-w.onHeadersErr:
		if err != nil {
			return err
		}
	case <-time.After(HeaderSyncTimeout):
		w.log.Warnf("Timed out waiting for the block header data")
		return fault.TimeoutWaitingForHeader
	}

	return nil
}

// QueryBlockDataByPeer will send GetData command to a peer
func (w *p2pWatcher) QueryBlockDataByPeer(p *peer.Peer, hash *chainhash.Hash) {
	blockDataMsg := wire.NewMsgGetData()
	blockDataMsg.AddInvVect(&wire.InvVect{
		Type: wire.InvTypeBlock,
		Hash: *hash,
	})

	p.QueueMessage(blockDataMsg, nil)
}

// lookupPayment will trigger a block re-scan process to check potential payments
// back to certains blocks
func (w *p2pWatcher) lookupPaymentFromPeer(p *peer.Peer, lookUpToHeight int32) {
	if lookUpToHeight == 0 {
		return
	}

	if p != nil {
		w.log.Infof("Look up payments by height from: %d, to: %d\n", lookUpToHeight, w.lastHeight)
		for h := w.lastHeight; h >= lookUpToHeight; h-- {
			hash, err := w.storage.GetHash(h)
			if err != nil {
				w.log.Errorf("unable to look up payment for height: %d. error: %s", h, err)
				return
			}

			w.log.Tracef("Fetch block data of block: %d %s", h, hash)
			w.QueryBlockDataByPeer(p, hash)
		}
	}
}

// fetchMoreAddress will fetch new messages from the bitcoin network
func (w *p2pWatcher) fetchMoreAddress() {
	w.Add(1)
	defer w.Done()
	for {
		select {
		case <-time.After(10 * time.Second):
			if w.addrManager.NeedMoreAddresses() {
				w.log.Debugf("Need more address. Fetch address from peers.")
				w.connectedPeers.Range(func(k string, p *peer.Peer) {
					p.QueueMessage(wire.NewMsgGetAddr(), nil)
				})
			}
		case <-w.shutdown:
			w.log.Trace("stop syncing…")
			return
		}
	}
}

// sleep is an internal function that could be interrupted by shutdown channel.
// It will return an error if the water is shutted down.
func (w *p2pWatcher) sleep(d time.Duration) error {
	select {
	case <-w.shutdown:
		return fault.ProcessStopping
	case <-time.After(d):
		return nil
	}
}

func (w *p2pWatcher) sync() {
	w.Add(1)
	defer w.Done()

	// This it the main loop that starts syncing loop and rollback if data is not consistent.
	for !w.stopping {
		// This is the loop for syncing process
	SYNC_LOOP:
		for {
			select {
			case <-w.shutdown:
				w.log.Trace("stop syncing…")
				return
			case p := <-w.getPeer():
				w.log.Infof("Peer block height: %d, our block height: %d", p.LastBlock(), w.lastHeight)
				err := w.syncHeaderFromPeer(p)

				if err != nil {
					switch err {
					case fault.NoNewBlockHeadersFromPeer:
						if p.LastBlock() < w.lastHeight {
							p.Disconnect()
						} else {
							w.log.Debug("no new headers. wait for next sync…")
							if err := w.sleep(30 * time.Second); err == fault.ProcessStopping {
								w.log.Trace("stop syncing…")
								return
							}
						}
					case fault.MissingPreviousBlockHeader:
						w.log.Warnf("Incorrect block data: %s", err)
						break SYNC_LOOP
					case fault.ProcessStopping:
						w.log.Trace("stop syncing…")
						return
					default:
						// The loop will retry the fetching process if there are other errors
						w.log.Warnf("fetch header error: %s", err)
					}
				} else {
					if p.LastBlock() <= w.lastHeight {
						if err := w.sleep(30 * time.Second); err == fault.ProcessStopping {
							w.log.Trace("stop syncing…")
							return
						}
					}
				}
			}
		}

		if err := w.rollbackBlock(); err != nil {
			w.log.Errorf("Fail to rollback blocks. Error: %s", err)
		}
	}
}

func (w *p2pWatcher) Run(args interface{}, shutdown <-chan struct{}) {
	w.log.Infof("last hash: %s, last block height: %d", w.lastHash, w.lastHeight)

	w.addrManager.Start()

	// add peer address by dns seed
outer:
	for _, seed := range w.networkParams.DNSSeeds {
		ips, err := net.LookupIP(seed.Host)
		if err != nil {
			w.log.Warnf("Fail to look up ip from DNS. Error: %s", err)
			continue outer
		}
	inner:
		for i, ip := range ips {
			// use DNS seed as a peer up to half of target outbound peer amounts
			if i > MaximumOutboundPeers/2 {
				break inner
			}
			if err := w.addrManager.AddAddressByIP(net.JoinHostPort(ip.String(), w.networkParams.DefaultPort)); err != nil {
				w.log.Warnf("Can not add an IP into address manager. Error: %s", err)
			}

		}
	}

	nodeCount := 0
bootstrap_loop:
	for _, hostPort := range w.bootstrapNodes {
		conn, err := net.Dial("tcp", hostPort)
		if err != nil {
			w.log.Warnf("Can not establish connection to nodes. Error: %s", err)
			continue bootstrap_loop
		}

		if _, err := w.peerNeogotiate(conn); err != nil {
			w.log.Warnf("Can not establish connection to nodes. Error: %s", err)
		} else {
			nodeCount += 1
		}
	}

	if len(w.bootstrapNodes) > 0 && nodeCount == 0 {
		logger.Panicf("unable to connect to any %s nodes", w.currency)
	}

	w.connManager.Start()

	go func() {
		for {
			w.log.Infof("Connected Peers: %d", w.connectedPeers.Len())
			// w.connectedPeers.Iter(func(k string, v *peer.Peer) {
			// 	w.log.Info("Peer Last Block:", v.LastBlock())
			// })
			time.Sleep(30 * time.Second)
		}
	}()

	go w.fetchMoreAddress()
	go w.sync()

	<-shutdown
	w.log.Info("shutting down…")
	w.StopAndWait()
	w.log.Info("stopped")
}

// StopAndWait will stop the watcher process and wait until all subroutines
// be terminated successfully.
func (w *p2pWatcher) StopAndWait() {
	if w.stopping {
		return
	}
	w.stopping = true
	close(w.shutdown)

	w.connManager.Stop()
	w.addrManager.Stop()

	if w.lastHeight == 0 || w.lastHeight == w.checkpoint.Height {
		return
	}

	checkpointHeight := checkpointBackLimit * (w.lastHeight / checkpointBackLimit)
	if err := w.storage.SetCheckpoint(checkpointHeight); err != nil {
		w.log.Errorf("Can not update the new check point. Error: %s", err)
	}

	w.Wait()
}

// getPeer will return a peer channel which will pipe a peer from
// connected peer randomly by the iteration of a map
// the channel will closed immediately right after a peer is successfully submitted
// Note: This is not a perfect random mechanism. But what we need is
// to have a way to have chances to get peers from different sources.
func (w *p2pWatcher) getPeer() <-chan *peer.Peer {
	peers := make(chan *peer.Peer)
	go func() {
		defer close(peers)
	loop:
		for {
			p := w.connectedPeers.First()
			if p == nil {
				time.Sleep(time.Second)
				continue loop
			}

			if w.lastHeight-p.LastBlock() > 100 {
				p.Disconnect()
				w.log.Tracef("Disconnect out-date peer: %s", p.Addr())
				time.Sleep(time.Second)
				continue loop
			}

			peers <- p
			break loop
		}
	}()
	return peers
}

// onPeerVerAck will be invoked right after a peer accepts our connection and will
// start responding our commands
func (w *p2pWatcher) onPeerVerAck(p *peer.Peer, msg *wire.MsgVerAck) {
	if w.connectedPeers.Exist(p.Addr()) {
		w.log.Tracef("Drop duplicated connection: %s", p.Addr())
		p.Disconnect()
		return
	}

	w.connectedPeers.Add(p.Addr(), p)
	w.addrManager.Good(p.NA())

	w.log.Tracef("Complete neogotiation with the peer: %s", p.Addr())
}

// onPeerAddr will add discovered new addresses into address manager
func (w *p2pWatcher) onPeerAddr(p *peer.Peer, msg *wire.MsgAddr) {
	for _, a := range msg.AddrList {
		w.log.Tracef("Receive new address: %s:%d. Peer service: %s", a.IP, a.Port, a.Services)
		w.addrManager.AddAddress(a, w.srcAddr)
	}
}

// onPeerHeaders handles messages from peer for updating header data
func (w *p2pWatcher) onPeerHeaders(p *peer.Peer, msg *wire.MsgHeaders) {
	var headersErr error
	defer func() {
		select {
		case w.onHeadersErr <- headersErr:
		case <-time.After(time.Second):
		}
	}()

	if len(msg.Headers) == 0 {
		headersErr = fault.NoNewBlockHeadersFromPeer
		return
	}

	var hasNewHeader bool
	var newHash chainhash.Hash
	var firstNewHeight, newHeight int32

loop:
	for _, h := range msg.Headers {
		newHash = h.BlockHash()
		nh, _ := w.storage.GetHeight(&newHash)
		if nh > 0 {
			// If there is any error here, the hash would be nil.
			// That means the height is not correctly set. We will do nothing and
			// assume it is a new hash
			if hash, _ := w.storage.GetHash(nh); hash != nil {
				if reflect.DeepEqual(hash.CloneBytes(), newHash.CloneBytes()) {
					w.log.Tracef("Omit the same hash: %s, %d", newHash, nh)
					if nh > w.lastHeight {
						w.lastHash = hash
						w.lastHeight = nh
					}
					continue loop
				}
			}
		}

		hasNewHeader = true

		// Since the header is new, we will then check whether its previous header is existed.
		// If a fork has happened then return an error
		// The event will break the sync loop and trigger a rollback process
		prevHeight, err := w.storage.GetHeight(&h.PrevBlock)
		if err != nil || prevHeight == -1 {
			p.Disconnect()
			headersErr = fault.MissingPreviousBlockHeader
			return
		}
		prevHash, err := w.storage.GetHash(prevHeight)
		if err != nil || !reflect.DeepEqual(prevHash.CloneBytes(), h.PrevBlock.CloneBytes()) {
			p.Disconnect()
			headersErr = fault.MissingPreviousBlockHeader
			return
		}

		newHeight = prevHeight + 1
		if firstNewHeight == 0 && time.Since(h.Timestamp) < PaymentExpiry {
			firstNewHeight = newHeight
		}

		w.log.Debugf("Add block hash: %s, %d", newHash, newHeight)
		if err := w.storage.StoreBlock(newHeight, &newHash); err != nil {
			headersErr = err
			return
		}
	}

	// check if there is any new headers at this processing
	if !hasNewHeader {
		headersErr = fault.NoNewBlockHeadersFromPeer
		return
	}

	if firstNewHeight > 0 {
		// TODO: look up from range instead of from the latest because there will be a race condition
		go w.lookupPaymentFromPeer(p, firstNewHeight)
	}

	if newHeight > p.LastBlock() {
		p.UpdateLastBlockHeight(newHeight)
		p.UpdateLastAnnouncedBlock(&newHash)
	}
	w.lastHash = &newHash
	w.lastHeight = newHeight
}

func (w *p2pWatcher) rollbackBlock() error {
	deleteDownTo := w.lastHeight - checkpointBackLimit

	// prevent from rolling back too much blocks
	if deleteDownTo < w.checkpoint.Height {
		deleteDownTo = w.checkpoint.Height
	}

	w.log.Infof("Start rolling back blocks to: %d", deleteDownTo)
	if err := w.storage.RollbackTo(w.lastHeight, deleteDownTo); err != nil {
		return err
	}

	lastHash, err := w.storage.GetHash(deleteDownTo)
	if err != nil {
		return err
	}

	w.lastHash = lastHash
	w.lastHeight = deleteDownTo
	return nil
}

func (w *p2pWatcher) examineTransaction(tx *wire.MsgTx) ([]byte, map[string]uint64) {
	var id []byte
	amounts := map[string]uint64{}

loop:
	for _, txout := range tx.TxOut {
		// if script starts with `6a30`, the rest of bytes would be a potential payment id
		index := bytes.Index(txout.PkScript, []byte{106, 48})
		if index == 0 {
			id = txout.PkScript[2:]
		} else {
			s, err := txscript.ParsePkScript(txout.PkScript)
			if err != nil {
				continue loop
			}

			addr, err := s.Address(w.networkParams)
			if err != nil {
				continue loop
			}
			amounts[addr.String()] = uint64(txout.Value)

			address2, err := litecoin.TransformAddress(addr.String())
			if nil == err && address2 != addr.String() {
				amounts[address2] = uint64(txout.Value)
			}

		}
	}

	return id, amounts
}

// onPeerBlock handles block messages from peer. It abstracts transactions from block data to
// collect all potential bitmark payment transactions.
func (w *p2pWatcher) onPeerBlock(p *peer.Peer, msg *wire.MsgBlock, buf []byte) error {
	blockHash := msg.BlockHash().String()

	if time.Since(msg.Header.Timestamp) > PaymentExpiry {
		return fault.BlockIsTooOld
	}

	hash := msg.BlockHash()
	blockHeight, _ := w.storage.GetHeight(&hash)
	if blockHeight == -1 {
		return fault.BlockHeaderNotFound
	}

	if _, found := w.blockCache.Get(blockHash); found {
		return fault.BlockAlreadyProcessed
	}

	for _, tx := range msg.Transactions {
		id, amounts := w.examineTransaction(tx)
		if id != nil {
			var payId pay.PayId
			copy(payId[:], id[:])
			txId := tx.TxHash().String()

			w.log.Debugf("Find a potential payment. payId: %s, txId: %s", payId.String(), txId)

			reservoir.SetTransferVerified(
				payId,
				&reservoir.PaymentDetail{
					Currency: w.currency,
					TxID:     txId,
					Amounts:  amounts,
				},
			)
		}
	}

	w.blockCache.Set(blockHash, true, 0)
	return nil
}

// peerConfig returns a payment template. The `ChainParams` will vary between
// different network settings in `p2pWatcher`.
func (w *p2pWatcher) peerConfig() *peer.Config {
	return &peer.Config{
		UserAgentName:    "bitmarkd-payment-lightclient",
		UserAgentVersion: "0.1.0",
		ChainParams:      w.networkParams,
		DisableRelayTx:   true,
		Services:         0,
		Listeners: peer.MessageListeners{
			OnVersion: func(p *peer.Peer, msg *wire.MsgVersion) *wire.MsgReject {
				return nil
			},
			OnVerAck:  w.onPeerVerAck,
			OnAddr:    w.onPeerAddr,
			OnHeaders: w.onPeerHeaders,
			OnBlock: func(p *peer.Peer, msg *wire.MsgBlock, buf []byte) {
				w.log.Tracef("on block: %s", msg.BlockHash())

				if err := w.onPeerBlock(p, msg, buf); err != nil {
					w.log.Tracef("ignore block processing for block: %s  error: %s", msg.BlockHash(), err)
				}
			},

			OnTx: func(p *peer.Peer, msg *wire.MsgTx) {
				w.log.Debugf("tx: %+v", msg)
			},
			OnAlert: func(p *peer.Peer, msg *wire.MsgAlert) {
				w.log.Debugf("alert: %+v", msg)
			},
			OnNotFound: func(p *peer.Peer, msg *wire.MsgNotFound) {
				w.log.Debugf("not found: %+v", msg)
			},
			OnReject: func(p *peer.Peer, msg *wire.MsgReject) {
				w.log.Debugf("reject: %+v", msg)
			},
		},
	}
}

// peerNeogotiate will neogotiate with the remote peer to complete the connection
func (w *p2pWatcher) peerNeogotiate(conn net.Conn) (*peer.Peer, error) {
	ipAddr := conn.RemoteAddr().String()
	p, err := peer.NewOutboundPeer(w.peerConfig(), ipAddr)
	if err != nil {
		return nil, err
	}
	w.addrManager.Connected(p.NA())

	w.log.Tracef("Try to associate connection to: %s", ipAddr)
	p.AssociateConnection(conn)

	return p, nil
}

// onConnectionConnected is callback function which is invoked by connection manager when
// a peer connection has successfully established.
func (w *p2pWatcher) onConnectionConnected(connReq *connmgr.ConnReq, conn net.Conn) {
	p, err := w.peerNeogotiate(conn)
	if err != nil {
		w.log.Warnf("Peer: %s neogotiation failed. Error: %s", connReq.Addr.String(), err)
		w.connManager.Disconnect(connReq.ID())
	}

	// To info connection manager that a connection is terminated
	go func() {
		p.WaitForDisconnect()
		w.connManager.Disconnect(connReq.ID())
	}()
}

// onConnectionDisconnected is callback function which is invoked by connection manager when
// one of its connection request is disconnected.
func (w *p2pWatcher) onConnectionDisconnected(connReq *connmgr.ConnReq) {
	w.log.Debugf("Clean up disconnected peer: %s", connReq.Addr.String())
	w.connectedPeers.Delete(connReq.Addr.String())
}
