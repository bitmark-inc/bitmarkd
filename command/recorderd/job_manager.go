// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

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

func newJobManager(calendar JobCalendar, proofer Proofer, rescheduleChannel <-chan struct{}, log *logger.L) JobManager {
	return &JobManagerData{
		calendar: calendar,
		proofer:  proofer,
		log:      log,
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
	for range j.channels.rescheduleChannel {
		j.log.Debug("receive reschedule event")
		if j.initialized {
			j.resetAllEvent()
		} else {
			j.initialized = true
			now := time.Now()
			j.calendar.RescheduleStartEventsPrior(now)
			j.calendar.RescheduleStopEventsPrior(now)
		}
		j.reschedule()
	}
	// j.log.Info("stop...")
}

func (j *JobManagerData) waitNextHasingStartEvent(duration time.Duration) {
	j.log.Infof("create goroutine for start events, next event duration: %.1f minutes",
		duration.Minutes())
	d := duration
	defer j.wg.Done()
loop:
	for {
		select {
		case <-time.After(d): // timeout
			j.log.Debugf("start hashing")
			j.proofer.StartHashing()
			now := time.Now()
			intf := j.calendar.PickNextStartEvent(now)
			j.calendar.RescheduleStartEventsPrior(now)

			if intf == nil {
				intf = j.calendar.PickNextStartEvent(now)
			}

			nextEvent := intf.(time.Time)
			d = j.timeDurationFromSrc2Dest(now, nextEvent)
			j.log.Infof("next start event at %s, duration: %.1f minutes",
				nextEvent.String(), d.Minutes())
		case <-j.channels.startEventChannel:
			j.log.Debug("reset event, terminate start goroutine...")
			break loop
		}
	}
	j.log.Infof("start event terminated...")
}

func (j *JobManagerData) waitNextHasingStopEvent(duration time.Duration) {
	j.log.Infof("create goroutine for stop events, next event duration: %.1f minutes",
		duration.Minutes())
	d := duration
	defer j.wg.Done()
loop:
	for {
		select {
		case <-time.After(d): // timeout
			j.log.Debug("stop hashing")
			j.proofer.StopHashing()
			now := time.Now()
			intf := j.calendar.PickNextStopEvent(now)
			j.calendar.RescheduleStopEventsPrior(now)

			if intf == nil {
				intf = j.calendar.PickNextStopEvent(now)
			}

			nextEvent := intf.(time.Time)
			d = j.timeDurationFromSrc2Dest(now, nextEvent)
			j.log.Infof("next stop event: %s, duration: %.1f minutes",
				nextEvent.String(), d.Minutes())
		case <-j.channels.stopEventChannel:
			j.log.Debug("reset event, terminate stop goroutine...")
			break loop
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
	intf := j.calendar.PickInitialiseStartEvent(now)
	nextEvent := intf.(time.Time)
	duration := nextEvent.Sub(now)
	go j.waitNextHasingStartEvent(duration)
}

func (j *JobManagerData) rescheduleStopEvent() {
	if j.calendar.RunForever() {
		return
	}
	now := time.Now()
	intf := j.calendar.PickInitialiseStopEvent(now)
	nextEvent := intf.(time.Time)
	duration := nextEvent.Sub(now)

	go j.waitNextHasingStopEvent(duration)
}

func (j *JobManagerData) timeDurationFromSrc2Dest(src time.Time, dest time.Time) time.Duration {
	return dest.Sub(src)
}
