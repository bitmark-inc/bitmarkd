package main

import (
	"runtime"
	"sync"
	"testing"

	"github.com/bitmark-inc/logger"
)

const (
	defaultActiveThreadCount = 0
)

var (
	stopChannel = make(chan struct{}, runtime.NumCPU())
)

func setupProofer(t *testing.T) *ProoferData {
	setupLogger(t)
	return &ProoferData{
		proofIDs:    make([]bool, runtime.NumCPU()),
		stopChannel: stopChannel,
		log:         logger.New("test"),
	}
}

func teardownProofer(p *ProoferData, threadCount uint32) {
	for i := uint32(0); i < threadCount; i++ {
		p.activeThreadDecrement(i)
	}
	logger.Finalise()
	removeLogFiles()
}

func TestActiveThreadCount(t *testing.T) {
	setupLogger(t)
	p := setupProofer(t)
	defer teardownProofer(p, defaultActiveThreadCount+1)

	threadCount := p.activeThread()
	if defaultActiveThreadCount != threadCount {
		t.Errorf("initial active thread count %d differs from %d",
			threadCount, defaultActiveThreadCount)
	}
	p.activeThreadIncrement(0)
	defer p.activeThreadDecrement(0)

	threadCount = p.activeThread()
	expected := uint32(defaultActiveThreadCount + 1)
	if expected != threadCount {
		t.Errorf("after increasing active thread, count %d different thatn expected %d",
			threadCount, expected)
	}
}

func TestNextProoferID(t *testing.T) {
	p := setupProofer(t)
	defer teardownProofer(p, defaultActiveThreadCount+1)

	totalCPU := runtime.NumCPU()

	expected := []struct {
		threadIncrementCount int
		expected             int
	}{
		{0, 0},                     // none created
		{1, 1},                     // additional 1 created, next ID will be 1
		{1, 2},                     // additional 1 created, next ID will be 2
		{totalCPU, errorProoferID}, // additional total cpu count created, next ID will be -1
	}

	for _, s := range expected {
		p.createProofer(uint32(s.threadIncrementCount))
		nextID, _ := p.nextProoferID()
		if s.expected != nextID {
			t.Errorf("error getting next proofer ID, expect %d but get %d", s.expected, nextID)
		}
	}
}

func TestProoferInitProoferIDs(t *testing.T) {
	setupLogger(t)
	p := setupProofer(t)
	defer teardownProofer(p, defaultActiveThreadCount+1)

	expected := runtime.NumCPU()
	actual := len(p.proofIDs)
	if actual != expected {
		t.Errorf("proofer init error, expect proofer ID slice size %d but get %d", expected, actual)
	}
}

func TestDifferenceToTargetThreadCount(t *testing.T) {
	cpuCount := 10
	p := &ProoferData{
		cpuCount: cpuCount,
	}
	expected := []struct {
		targetThreadCount  uint32
		currentThreadCount uint32
		output             int32
	}{
		{uint32(5), uint32(4), int32(1)},
		{uint32(2), uint32(2), int32(0)},
		{uint32(1), uint32(2), int32(-1)},
		{uint32(6), uint32(1), int32(5)},
		{uint32(4), uint32(8), int32(-4)},
		{uint32(100), uint32(200), int32(-cpuCount + 1)},
		{uint32(100), uint32(50), int32(cpuCount)},
	}

	for i, s := range expected {
		output := p.differenceToTargetThreadCount(s.targetThreadCount, s.currentThreadCount)
		if output != s.output {
			t.Errorf("%dth test, error get thread increment value, expect %d but get %d",
				i, s.output, output)
		}

	}
}

func TestSetWorking(t *testing.T) {
	p := &ProoferData{}
	fixture := []struct {
		working  bool
		expected bool
	}{
		{true, true},
		{false, false},
	}

	for _, s := range fixture {
		p.setWorking(s.working)
		actual := p.workingNow
		if actual != s.expected {
			t.Errorf("error set working status, expect %t but get %t",
				s.expected, actual)
		}
	}
}

func TestDeleteProofer(t *testing.T) {
	p := setupProofer(t)
	defer teardownProofer(p, defaultActiveThreadCount+1)

	var wg sync.WaitGroup
	count := 2
	received := 0
	wg.Add(count)

	go func() {
		for i := 0; i < count; i++ {
			select {
			case <-p.stopChannel:
				received++
				wg.Done()
			}
		}
	}()
	go p.deleteProofer(int32(count))
	wg.Wait()
}
