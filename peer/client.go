// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package peer

import (
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/mode"
	"time"
)

// number of retries before giving up
const (
	initialPollDelay = 1 * time.Minute
	regularPollDelay = 5 * time.Minute
)

// the client thread
func (peer *peerData) client(t *thread) {

	t.log.Info("starting…")

	server := peer.server
	delay := time.After(initialPollDelay)
loop:
	for {
		select {
		case <-t.stop:
			break loop
		case <-delay:
			delay = time.After(regularPollDelay)
		}

		t.log.Info("loop")

		if 0 == server.ConnectionCount() {
			t.log.Info("no peers responding")
			continue loop
		}

		for i, a := range server.ActiveConnections() {
			t.log.Infof("active[%d] = %q", i, a)
		}

	getBlocks:
		for {
			t.log.Info("getBlocks")
			select {
			case <-t.stop:
				break loop
			default:
			}
			if highest, from, ok := highestBlockNumber(server, t.log); ok {
				t.log.Infof("highest bn: %d  from: %q", highest, from)
				n := block.Number() // the number of block being mined
				if highest >= n {   // equal because we need block 'n' if it is available
					t.log.Infof("resynchronise blocks: %d → %d from: %)", n, highest, from)
					mode.Set(mode.Resynchronise)
					t.resynchronise(server, n, highest, []string{from})
					continue getBlocks
				} else if mode.Is(mode.Resynchronise) {
					t.log.Info("normal")
					mode.Set(mode.Normal)
					peer.rebroadcast = true
				}
			}
			break getBlocks
		}

		if peer.rebroadcast {
			peer.rebroadcast = false
			t.rebroadcastTransactions(server)
		}
	}

	t.log.Info("shutting down…")
	t.log.Flush()

	close(t.done)
}
