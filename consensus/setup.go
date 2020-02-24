package consensus

import (
	"sync"
	"time"

	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/p2p"
	"github.com/bitmark-inc/logger"
)

var globalData Consensus

const (
	votingMetricRunInitial  = 60 * time.Second // should reference announce and p2p initial
	votingMetricRunInterval = 30 * time.Second
	machineRunInitial       = 70 * time.Second // should reference announce and p2p initial
)

//Consensus is a wrap struct for consensus  state machine
type Consensus struct {
	sync.RWMutex            // to allow locking
	Log           *logger.L // logger
	Node          *p2p.Node
	machine       Machine
	votingMetrics MetricsPeersVoting
	initialised   bool
	background    *background.T
}

// Initialise consensus package
func Initialise(node *p2p.Node, fastsync bool) error {
	globalData.Lock()
	defer globalData.Unlock()
	if globalData.initialised {
		return fault.AlreadyInitialised
	}

	if nil == node || nil == node.Host {
		panic("give an empty node or a empty host")
	}
	globalData.Log = logger.New("consensus")
	globalData.votingMetrics = NewMetricsPeersVoting(node)
	globalData.Log.Info("starting…")
	globalData.machine = NewConsensusMachine(node, &globalData.votingMetrics, fastsync)
	globalData.Log.Info("start background…")

	processes := background.Processes{
		&globalData.machine,
		&globalData.votingMetrics,
	}

	globalData.background = background.Start(processes, globalData.Log)
	return nil
}

// Finalise - stop all background tasks
func Finalise() error {
	if !globalData.initialised {
		return fault.NotInitialised
	}
	globalData.Log.Info("shutting down…")
	globalData.Log.Flush()

	// stop background
	globalData.background.Stop()
	// finally...
	globalData.initialised = false
	globalData.Log.Info("finished")
	globalData.Log.Flush()

	return nil
}
