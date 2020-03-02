package consensus

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/bitmarkd/p2p"
	"github.com/bitmark-inc/logger"
)

func TestMain(m *testing.M) {
	curPath := os.Getenv("PWD")
	var logConfig = logger.Configuration{
		Directory: curPath,
		File:      "consensus.log",
		Size:      1048576,
		Count:     20,
		Console:   false,
		Levels: map[string]string{
			logger.DefaultTag: "trace",
		},
	}
	if err := logger.Initialise(logConfig); err != nil {
		panic(fmt.Sprintf("logger initialization failed: %s", err))
	}
	globalData.machine.log = logger.New("consensus")
	os.Exit(m.Run())
}

func TestMachineState(t *testing.T) {
	states := []string{
		"Connecting",
		"HighestBlock",
		"ForkDetect",
		"FetchBlocks",
		"Rebuild",
		"Sampling",
		"*Unknown*",
	}
	cStateException := state(100)
	assert.Equal(t, states[0], cStateConnecting.String(), fmt.Sprintf("cStateConnecting does not return %v", states[0]))
	assert.Equal(t, states[1], cStateHighestBlock.String(), fmt.Sprintf("cStateHighestBlock does not return %v", states[1]))
	assert.Equal(t, states[2], cStateForkDetect.String(), fmt.Sprintf("cStateForkDetect does not return %v", states[2]))
	assert.Equal(t, states[3], cStateFetchBlocks.String(), fmt.Sprintf("cStateFetchBlocks does not return %v", states[3]))
	assert.Equal(t, states[4], cStateRebuild.String(), fmt.Sprintf("cStateRebuild does not return %v", states[4]))
	assert.Equal(t, states[5], cStateSampling.String(), fmt.Sprintf("cStateSampling does not return %v", states[5]))
	assert.Equal(t, states[6], cStateException.String(), fmt.Sprintf("cStateException does not return %v", states[6]))
}

func TestGetBlockHeight(t *testing.T) {
	newMachine := NewConsensusMachine(&p2p.Node{}, &MetricsPeersVoting{}, false)
	newMachine.targetHeight = 100
	globalData.machine = newMachine
	assert.Equal(t, newMachine.targetHeight, BlockHeight(), fmt.Sprintf("TestGetBlockHeight does not return %d", newMachine.targetHeight))
}
