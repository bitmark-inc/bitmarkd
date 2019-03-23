package main

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/bitmark-inc/logger"
)

const (
	defaultDelay     = time.Duration(1) * time.Microsecond
	defaultCheckTime = time.Duration(1) * time.Millisecond
)

var channels JobManagerChannel
var hashingCalled bool
var managerLogging logger.Configuration

type FakeProofer struct{}

func (p *FakeProofer) startHashing() {
	hashingCalled = true
}

func (p *FakeProofer) stopHashing() {
	hashingCalled = true
}

func (p *FakeProofer) refresh() {}

func (p *FakeProofer) isWorking() bool {
	return true
}

type FakeCalendar struct{}

func (c *FakeCalendar) pickNextStartEvent(t time.Time) interface{} {
	return time.Now()
}
func (c *FakeCalendar) pickNextStopEvent(t time.Time) interface{} {
	return time.Now()
}
func (c *FakeCalendar) pickInitialiseStartEvent(time.Time) interface{} {
	return time.Now()
}
func (c *FakeCalendar) pickInitialiseStopEvent(time.Time) interface{} {
	return time.Now()
}
func (c *FakeCalendar) refresh(ConfigCalendar)               {}
func (c *FakeCalendar) rescheduleStartEventsPrior(time.Time) {}
func (c *FakeCalendar) rescheduleStopEventsPrior(time.Time)  {}
func (c *FakeCalendar) setLog(*logger.L)                     {}
func (c *FakeCalendar) runForever() bool {
	return true
}

func setupTestJobManager() *JobManagerData {
	p := setupProoferInterface()
	channels.rescheduleChannel = make(chan struct{})
	channels.startEventChannel = make(chan struct{}, 1)
	channels.stopEventChannel = make(chan struct{}, 1)
	setupTestManagerLogger()

	j := &JobManagerData{
		calendar: &FakeCalendar{},
		proofer:  p,
		channels: JobManagerChannel{
			rescheduleChannel: channels.rescheduleChannel,
			startEventChannel: channels.startEventChannel,
			stopEventChannel:  channels.stopEventChannel,
		},
		log: logger.New("test"),
	}
	return j
}

func setupTestManagerLogger() {
	_ = os.Mkdir(logDirectory, 0770)
	managerLogging = loggerConfiguration()
	_ = logger.Initialise(managerLogging)
}

func teardownManager() {
	logger.Finalise()
	removeTestFiles()
}

func setupProoferInterface() Proofer {
	p := &FakeProofer{}
	return p
}

func TestWaitNextHashingStartEvent(t *testing.T) {
	j := setupTestJobManager()
	defer teardownManager()

	j.initialized = true
	hashingCalled = false
	j.wg.Add(1)
	go j.waitNextHasingStartEvent(defaultDelay)
	time.Sleep(defaultCheckTime)
	j.channels.startEventChannel <- struct{}{}
	j.wg.Wait()

	if !hashingCalled {
		t.Errorf("proofer start hashing is not called.")
	}
}

func TestWaitNextHashingStopEvent(t *testing.T) {
	j := setupTestJobManager()
	defer teardownManager()

	j.initialized = true
	hashingCalled = false
	j.wg.Add(1)
	go j.waitNextHasingStopEvent(defaultDelay)
	time.Sleep(defaultCheckTime)
	j.channels.stopEventChannel <- struct{}{}
	j.wg.Wait()

	if !hashingCalled {
		t.Errorf("proofer start hashing is not called.")
	}
}

func TestResetAllEvent(t *testing.T) {
	j := setupTestJobManager()
	defer teardownManager()

	received := 0
	goRoutinCount := 2
	var wg sync.WaitGroup
	wg.Add(goRoutinCount)
	go func() {
		for i := 0; i < goRoutinCount; i++ {
			select {
			case <-j.channels.startEventChannel:
				received++
				wg.Done()
			case <-j.channels.stopEventChannel:
				received++
				wg.Done()
			}

		}
	}()
	go j.resetAllEvent()
	wg.Wait()

	if received != goRoutinCount {
		t.Errorf("reset all event signal is not received")
	}
}

func TestTimeDurationFromSrc2Dest(t *testing.T) {
	j := setupTestJobManager()
	defer teardownManager()

	now := time.Now()
	fixture := []struct {
		src      time.Time
		expected time.Duration
	}{
		{now, time.Duration(10) * time.Minute},
		{now, time.Duration(100) * time.Hour},
	}

	for i, s := range fixture {
		actual := j.timeDurationFromSrc2Dest(now, now.Add(s.expected))
		if actual != s.expected {
			t.Errorf("%dth test fail, wrong time duration", i)
			t.Errorf("duration of %s to %s",
				stringifyTime(now), stringifyTime(now.Add(s.expected)))
			t.Errorf("expect %s but get %s", s.expected, actual)
		}
	}
}
