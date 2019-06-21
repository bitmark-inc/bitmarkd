// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package background_test

import (
	"fmt"
	"time"

	"github.com/bitmark-inc/bitmarkd/background"
)

type theState struct {
	count int
}

func Example() {

	proc := &theState{
		count: 10,
	}

	// list of background processes to start
	processes := background.Processes{
		proc,
	}

	p := background.Start(processes, nil)
	time.Sleep(time.Second)
	p.Stop()
}

func (state *theState) Run(args interface{}, shutdown <-chan struct{}) {

	fmt.Printf("initialise\n")

loop:
	for {
		select {
		case <-shutdown:
			break loop
		default:
		}

		state.count += 1
		time.Sleep(time.Millisecond)
	}

	fmt.Printf("finalise\n")
}
