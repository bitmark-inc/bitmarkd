package statemachine

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

// StateMachine a block state machine:
type StateMachine struct {
	log *logger.L
	state
}

// NewStateMachine get a new StateMachine
func NewStateMachine() StateMachine {
	machine := StateMachine{log: logger.New("machine")}
	return machine
}

//Run Run A StateMachine
func (m *StateMachine) Run(args interface{}, shutdown <-chan struct{}) {
	log := m.log
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
			m.start()
		}
	}
	log.Info("shutting down…")
	// TODO:  do close of stream
	//conn.destroy()
	log.Info("stopped")
}
func (m *StateMachine) start() {
	for m.stepTransition() {
	}
}

func (m *StateMachine) stepTransition() bool {
	log := m.log

	log.Infof("current state: %s", m.state)

	continueLooping := true

	switch m.state {
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
