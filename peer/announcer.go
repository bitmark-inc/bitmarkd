// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package peer

import (
	"github.com/bitmark-inc/bilateralrpc"
	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/gnomon"
	"time"
)

// various constants
const (
	announceCount = 20 // announcements per cycle
	announceTime  = 2 * time.Minute
)

// loop to read queue
func (peer *peerData) announcer(t *thread) {

	t.log.Info("starting…")

	server := peer.server

	// for sending out RPCs
	rpcCursor := &gnomon.Cursor{}

loop:
	for {
		select {
		case <-t.stop:
			break loop
		case <-time.After(announceTime):
			t.announceToAll(server, &rpcCursor)
		}
	}

	t.log.Info("shutting down…")
	t.log.Flush()

	close(t.done)
}

// announce my RPC ports
func (t *thread) announceToAll(server *bilateralrpc.Bilateral, rpcCursor **gnomon.Cursor) {

	peers, nextStart, err := announce.RecentPeers(*rpcCursor, announceCount, announce.TypeRPC)
	if nil != err {
		t.log.Errorf("recent peers: error: %v", err)
		return
	}

	for _, d := range peers {
		recent := d.(announce.RecentData)
		t.announceOne(server, &recent)
	}
	*rpcCursor = nextStart
}

func (t *thread) announceOne(server *bilateralrpc.Bilateral, recent *announce.RecentData) {

	t.log.Warnf("announce rpc at: %s", recent.Address)

	putArguments := RpcPutArguments{
		Address:     recent.Address,
		Fingerprint: *recent.Data.Fingerprint,
	}

	var putResult []struct {
		From  string
		Reply RpcPutReply
		Err   error
	}

	if err := server.Call(bilateralrpc.SendToAll, "RPCs.Put", &putArguments, &putResult, 0); nil != err {
		// if remote does not accept it is not really a problem for this node - just warn
		t.log.Warnf("RPCs.Put err = %v", err)
		return
	}

	// determine which node want the certificate
	to := make([]string, 0, len(putResult))
	for _, r := range putResult {
		if r.Reply.NeedCertificate {
			t.log.Infof("peer: %q needs certificate", r.From)
			to = append(to, r.From)
		}
	}

	if 0 == len(to) {
		return
	}

	certificate, found := announce.GetCertificate(recent.Data.Fingerprint)
	if !found {
		// should have certificate
		t.log.Errorf("missing certificate for: %v", recent.Data.Fingerprint)
		return
	}

	putCertArguments := PutCertificateArguments{
		Certificate: certificate,
	}

	if err := server.Cast(to, "Certificate.Put", &putCertArguments); nil != err {
		// if remote does not accept it is not really a problem for this node - just warn
		t.log.Warnf("Certificate.Put err = %v", err)
	}

	// send again to those that wanted certificate first
	if err := server.Cast(to, "RPCs.Put", &putArguments); nil != err {
		// if remote does not accept it is not really a problem for this node - just warn
		t.log.Warnf("RPCs.Put err = %v", err)
	}

}
