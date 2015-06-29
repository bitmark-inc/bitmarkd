// Copyright (c) 2014-2015 Bitmark Inc.
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
	resynchroniseAttempts = 3
	initialPollDelay      = 2 * time.Minute
	regularPollDelay      = 5 * time.Minute
)

// the client thread
func (peer *peerData) client(t *thread) {

	t.log.Info("starting…")

	server := peer.server
	retries := resynchroniseAttempts
	delay := initialPollDelay
loop:
	for {
		select {
		case <-t.stop:
			break loop
		case <-time.After(delay):
			delay = regularPollDelay
		}

		t.log.Info("loop")
		for i, a := range server.ActiveConnections() {
			t.log.Infof("active[%d] = %q", i, a)
		}

	getBlocks:
		for {
			t.log.Infof("getBlocks retries left: %d", retries)
			select {
			case <-t.stop:
				break loop
			default:
			}
			if highest, from, ok := highestBlockNumber(server, t.log); ok {
				t.log.Infof("highest bn = %d  from: %q", highest, from)
				retries = resynchroniseAttempts
				n := block.Number() // the number of block being mined
				if highest >= n {   // equal because we need block 'n' if it is available
					mode.Set(mode.Resynchronise)
					t.resynchronise(server, n, highest, []string{from})
					continue getBlocks
				}
				if mode.Is(mode.Resynchronise) {
					t.log.Infof("normal")
					mode.Set(mode.Normal)
				}
			} else {
				retries -= 1
				if retries <= 0 {
					// stand alone mode
					if mode.Is(mode.Resynchronise) {
						t.log.Infof("stand-alone")
						mode.Set(mode.Normal)
					}
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
