package main

import (
	"sync"
	"time"

	"github.com/bitmark-inc/logger"
)

type JobManager interface {
	Start()
}

const (
	JobManagerPrefix = "job-manager"
)

type JobManagerChannel struct {
	rescheduleChannel <-chan struct{}
	startEventChannel chan struct{}
	stopEventChannel  chan struct{}
}

type JobManagerData struct {
	calendar    JobCalendar
	proofer     Proofer
	channels    JobManagerChannel
	log         *logger.L
	initialized bool
	wg          sync.WaitGroup
}

func newJobManager(calendar JobCalendar, proofer Proofer, rescheduleChannel <-chan struct{}, logger *logger.L) JobManager {
	return &JobManagerData{
		calendar: calendar,
		proofer:  proofer,
		log:      logger,
		channels: JobManagerChannel{
			rescheduleChannel: rescheduleChannel,
			startEventChannel: make(chan struct{}, 1),
			stopEventChannel:  make(chan struct{}, 1),
		},
		initialized: false,
	}
}

func (j *JobManagerData) waitForRefresh() {
	j.log.Info("waiting to reschedule events")
	for {
		select {
		case <-j.channels.rescheduleChannel:
			j.log.Debug("receive reschedule event")
			if j.initialized {
				j.resetAllEvent()
			} else {
				j.initialized = true
				now := time.Now()
				j.calendar.rescheduleStartEventsPrior(now)
				j.calendar.rescheduleStopEventsPrior(now)
			}
			j.reschedule()
		}
	}
	j.log.Info("stop...")
}

func (j *JobManagerData) waitNextHasingStartEvent(duration time.Duration) {
	j.log.Infof("create goroutine for start events, next event duration: %.1f minutes",
		duration.Minutes())
	d := duration
	defer j.wg.Done()
	for {
		select {
		case <-time.After(d):
			j.log.Debugf("start hashing")
			j.proofer.startHashing()
			now := time.Now()
			intf := j.calendar.pickNextStartEvent(now)
			j.calendar.rescheduleStartEventsPrior(now)

			if nil == intf {
				intf = j.calendar.pickNextStartEvent(now)
			}

			nextEvent := intf.(time.Time)
			d = j.timeDurationFromSrc2Dest(now, nextEvent)
			j.log.Infof("next start event at %s, duration: %.1f minutes",
				nextEvent.String(), d.Minutes())
		case <-j.channels.startEventChannel:
			j.log.Debug("reset event, terminate start goroutine...")
			return
		}
	}
	j.log.Infof("start event terminated...")
}

func (j *JobManagerData) waitNextHasingStopEvent(duration time.Duration) {
	j.log.Infof("create goroutine for stop events, next event duration: %.1f minutes",
		duration.Minutes())
	d := duration
	defer j.wg.Done()
	for {
		select {
		case <-time.After(d):
			j.log.Debug("stop hashing")
			j.proofer.stopHashing()
			now := time.Now()
			intf := j.calendar.pickNextStopEvent(now)
			j.calendar.rescheduleStopEventsPrior(now)

			if nil == intf {
				intf = j.calendar.pickNextStopEvent(now)
			}

			nextEvent := intf.(time.Time)
			d = j.timeDurationFromSrc2Dest(now, nextEvent)
			j.log.Infof("next stop event: %s, duration: %.1f minutes",
				nextEvent.String(), d.Minutes())
		case <-j.channels.stopEventChannel:
			j.log.Debug("reset event, terminate stop goroutine...")
			return
		}
	}
	j.log.Infof("stop event terminated...")
}

func (j *JobManagerData) resetAllEvent() {
	j.log.Infof("reset all events...")
	j.wg.Add(2)
	j.channels.startEventChannel <- struct{}{}
	j.channels.stopEventChannel <- struct{}{}
	j.wg.Wait()
	return
}

func (j *JobManagerData) Start() {
	go j.waitForRefresh()
}

func (j *JobManagerData) reschedule() {
	j.log.Debug("reschedule...")
	j.rescheduleStartEvent()
	j.rescheduleStopEvent()
}

func (j *JobManagerData) rescheduleStartEvent() {
	now := time.Now()
	intf := j.calendar.pickInitialiseStartEvent(now)
	nextEvent := intf.(time.Time)
	duration := nextEvent.Sub(now)
	go j.waitNextHasingStartEvent(duration)
}

func (j *JobManagerData) rescheduleStopEvent() {
	if j.calendar.runForever() {
		return
	}
	now := time.Now()
	intf := j.calendar.pickInitialiseStopEvent(now)
	nextEvent := intf.(time.Time)
	duration := nextEvent.Sub(now)

	go j.waitNextHasingStopEvent(duration)
}

func (j *JobManagerData) timeDurationFromSrc2Dest(src time.Time, dest time.Time) time.Duration {
	return dest.Sub(src)
}
