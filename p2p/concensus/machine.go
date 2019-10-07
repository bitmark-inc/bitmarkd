package concensus

import (
	"time"

	"github.com/bitmark-inc/logger"
)

// various timeouts
const (
	// pause to limit bandwidth
	cycleInterval = 15 * time.Second

	// time out for connections
	connectorTimeout = 60 * time.Second

	// number of cycles to be 1 block out of sync before resync
	samplelingLimit = 10

	// number of blocks to fetch in one set
	fetchBlocksPerCycle = 200

	// fail to fork if height difference is greater than this
	forkProtection = 60

	// do not proceed unless this many clients are connected
	minimumClients = 5

	// total number of dynamic clients
	maximumDynamicClients = 25

	// client should exist at least 1 response with in this number
	activePastSec = 60
)

// ConcensusMachine a block state machine:
type ConcensusMachine struct {
	log *logger.L
	state
}

// NewStateMachine get a new StateMachine
func NewStateMachine() ConcensusMachine {
	machine := ConcensusMachine{log: logger.New("machine")}
	return machine
}

//Run Run A ConcensusMachine
func (c *ConcensusMachine) Run(args interface{}, shutdown <-chan struct{}) {
	log := c.log
	log.Info("starting block state machine…")
	timer := time.After(cycleInterval)
loop:
	for {
		// wait for shutdown
		log.Debug("waiting…")

		select {
		case <-shutdown:
			break loop
		case <-timer: // timer has priority over queue
			timer = time.After(cycleInterval)
			c.start()
		}
	}
	log.Info("shutting down…")
	// TODO:  do close of stream
	//conn.destroy()
	log.Info("stopped")
}
func (c *ConcensusMachine) start() {
	for c.stepTransition() {
	}
}

func (c *ConcensusMachine) stepTransition() bool {
	log := c.log

	log.Infof("current state: %s", c.state)

	continueLooping := true

	switch c.state {
	case cStateConnecting:
		continueLooping = false

	case cStateHighestBlock:
	case cStateForkDetect:
	case cStateFetchBlocks:
		continueLooping = false
	case cStateRebuild:
	case cStateSampling:
	}
	return continueLooping
}
