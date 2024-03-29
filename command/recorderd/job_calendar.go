// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/bitmark-inc/logger"
)

const (
	defaultNum                = 0
	defaultTimeStrErrorMsg    = "clock time out of range"
	defaultTimePeriodErrorMsg = "time period format error"
	timePeriodSeparator       = ","
	clockSeparator            = "-"
	spaceChar                 = " "
	allDayClockStr            = ""
	defaultIndex              = 0
	oneWeekDuration           = time.Duration(24*7) * time.Hour
	delayOfStartStop          = time.Duration(5) * time.Second
	jobCalendarPrefix         = "calendar"
)

type JobCalendar interface {
	PickNextStartEvent(time.Time) interface{}
	PickNextStopEvent(time.Time) interface{}
	PickInitialiseStartEvent(time.Time) interface{}
	PickInitialiseStopEvent(time.Time) interface{}
	Refresh(calendar ConfigCalendar)
	RescheduleStartEventsPrior(event time.Time)
	RescheduleStopEventsPrior(event time.Time)
	RunForever() bool
	SetLog(l *logger.L)
}

type NumberRange struct {
	min uint32
	max uint32
}

// collapse events, all start time put in one place, all end time put in one place
type FlattenEvents struct {
	start []time.Time
	stop  []time.Time
}

type SingleEvent struct {
	start time.Time
	stop  time.Time
}

type JobCalendarData struct {
	flattenEvents     FlattenEvents
	events            map[time.Weekday][]SingleEvent
	rawData           ConfigCalendar
	rescheduleChannel chan<- struct{}
	log               *logger.L
}

type TimeData struct {
	hour, minute uint32
}

func newJobCalendar(channel chan<- struct{}) JobCalendar {
	return &JobCalendarData{
		flattenEvents: FlattenEvents{
			start: []time.Time{},
			stop:  []time.Time{},
		},
		events: map[time.Weekday][]SingleEvent{
			time.Sunday:    {},
			time.Monday:    {},
			time.Tuesday:   {},
			time.Wednesday: {},
			time.Thursday:  {},
			time.Friday:    {},
			time.Saturday:  {},
		},
		rawData:           ConfigCalendar{},
		rescheduleChannel: channel,
	}
}

func (j *JobCalendarData) newEmptyFlattenEvents() FlattenEvents {
	return FlattenEvents{
		start: []time.Time{},
		stop:  []time.Time{},
	}
}

func (j *JobCalendarData) newEmptyEvents() map[time.Weekday][]SingleEvent {
	return map[time.Weekday][]SingleEvent{
		time.Sunday:    {},
		time.Monday:    {},
		time.Tuesday:   {},
		time.Wednesday: {},
		time.Thursday:  {},
		time.Friday:    {},
		time.Saturday:  {},
	}
}

func (j *JobCalendarData) SetLog(l *logger.L) {
	j.log = l
}

func (j *JobCalendarData) RunForever() bool {
	return len(j.flattenEvents.stop) == 0
}

func (j *JobCalendarData) Refresh(calendar ConfigCalendar) {
	j.log.Debug("refresh calendar")
	if !j.isSameCalendar(calendar) {
		j.log.Debug("calendar change")
		j.setNewCalendar(calendar)
		j.resetEvents()
		j.parseRawData(calendar)
		j.removeRedundantStopEvent()
		j.printEvents()
		j.notifyJobManager()
	}
}

func (j *JobCalendarData) resetEvents() {
	j.flattenEvents = j.newEmptyFlattenEvents()
	j.events = j.newEmptyEvents()
}

func (j *JobCalendarData) notifyJobManager() {
	j.log.Debug("notify manager for new calendar settings...")
	j.rescheduleChannel <- struct{}{}
}

func (j *JobCalendarData) setNewCalendar(calendar ConfigCalendar) {
	j.rawData = calendar
}

func (j *JobCalendarData) parseRawData(calendar ConfigCalendar) {
	j.convertWeekScheduleToEvents()
	j.sortFlattenEventsFromEarlier2Later()
}

func (j *JobCalendarData) removeRedundantStopEvent() {
	j.log.Debug("removing redundant events...")
	start := j.flattenEvents.start
	stop := j.flattenEvents.stop
	redundantIdx := make([]bool, len(j.flattenEvents.stop))

loop:
	for i, k := 0, 0; i < len(start) && k < len(stop); {
		if start[i].Equal(stop[k]) {
			j.log.Debugf("%+v stop event is redundant", j.flattenEvents.stop[k])
			redundantIdx[k] = true
			i++
			k++
			continue loop
		}
		if start[i].After(stop[k]) {
			k++
		} else {
			i++
		}
	}
	newSlice := make([]time.Time, 0, len(j.flattenEvents.stop))
	for i := 0; i < len(redundantIdx); i++ {
		if !redundantIdx[i] {
			newSlice = append(newSlice, stop[i])
		}
	}
	j.flattenEvents.stop = newSlice
}

func isEventAlreadyExist(times []time.Time, event time.Time) (bool, int) {
	if len(times) == 0 {
		return false, defaultIndex
	}
	for i, v := range times {
		if v == event {
			return true, i
		}
	}
	return false, defaultIndex
}

func isSameTime(t1 TimeData, t2 TimeData) bool {
	if t1.hour == t2.hour && t1.minute == t2.minute {
		return true
	}
	return false
}

func isTimeDataFirstEarlierThanSecond(first TimeData, second TimeData) bool {
	if first.hour < second.hour {
		return true
	}
	if first.minute < second.minute {
		return true
	}
	return false
}

func (j *JobCalendarData) isTimeBooked(event time.Time) bool {
	weekDay := event.Weekday()
	events := j.events[weekDay]

	for _, t := range events {
		afterOrEqualToStartTime := t.start.Before(event) || t.start.Equal(event)
		beforeOrEqualToEndTime := t.stop.After(event) || t.stop.Equal(event)

		if afterOrEqualToStartTime && beforeOrEqualToEndTime {
			return true
		}

		if events[0].stop.IsZero() && afterOrEqualToStartTime {
			return true
		}
	}
	return false
}

func (j *JobCalendarData) PickInitialiseStartEvent(event time.Time) interface{} {
	if j.isTimeBooked(event) {
		j.log.Debugf("working time, start after %s", delayOfStartStop.String())
		return event.Add(delayOfStartStop)
	}
	return j.PickNextStartEvent(event)
}

func (j *JobCalendarData) PickInitialiseStopEvent(event time.Time) interface{} {
	if j.isTimeBooked(event) {
		return j.PickNextStopEvent(event)
	}
	j.log.Debugf("not working time, stop after %s", delayOfStartStop.String())
	return event.Add(delayOfStartStop)
}

func (j *JobCalendarData) PickNextStartEvent(event time.Time) interface{} {
	for _, e := range j.flattenEvents.start {
		if e.After(event) {
			j.log.Infof("next start event at %s", e)
			return e
		}
	}
	j.log.Error("cannot find next start event")
	j.printEvents()
	return nil
}

func (j *JobCalendarData) PickNextStopEvent(event time.Time) interface{} {
	for _, e := range j.flattenEvents.stop {
		if e.After(event) {
			j.log.Infof("next stop event at %s", e)
			return e
		}
	}
	j.log.Info("cannot find next stop event")
	j.printEvents()
	return nil
}

func (j *JobCalendarData) RescheduleStartEventsPrior(event time.Time) {
	if len(j.flattenEvents.start) == 0 || j.flattenEvents.start[0].After(event) {
		return
	}
	times := j.flattenEvents.start
	newSlices := make([]time.Time, 0, len(times))
	schedules := make([]time.Time, 0, len(times))
loop:
	for i, t := range times {
		if t.Before(event) || t.Equal(event) {
			schedules = append(schedules, t.Add(oneWeekDuration))
		} else {
			newSlices = append(newSlices, times[i:]...)
			break loop
		}
	}
	newSlices = append(newSlices, schedules...)
	j.flattenEvents.start = newSlices
}

func (j *JobCalendarData) RescheduleStopEventsPrior(event time.Time) {
	if len(j.flattenEvents.stop) == 0 || j.flattenEvents.stop[0].After(event) {
		return
	}
	times := j.flattenEvents.stop
	newSlices := make([]time.Time, 0, len(times))
	schedules := make([]time.Time, 0, len(times))
loop:
	for i, t := range times {
		if t.Before(event) || t.Equal(event) {
			schedules = append(schedules, t.Add(oneWeekDuration))
		} else {
			newSlices = append(newSlices, times[i:]...)
			break loop
		}
	}
	newSlices = append(newSlices, schedules...)
	j.flattenEvents.stop = newSlices
}

func (j *JobCalendarData) removeEventFrom(times []time.Time, event time.Time) ([]time.Time, error) {
	exist, idx := isEventAlreadyExist(times, event)
	if !exist {
		return times, nil
	}
	return append(times[:idx], times[idx+1:]...), nil
}

func (j *JobCalendarData) weekDayCurrent2Target(current time.Weekday, target time.Weekday) int {
	return int(target) - int(current)
}

func (j *JobCalendarData) parseClockStr(clock string) (TimeData, error) {
	if clock == "24:00" || clock == "24:0" {
		return TimeData{
			hour:   uint32(24),
			minute: uint32(0),
		}, nil
	}

	t, err := time.Parse("15:04", clock)
	if err != nil {
		j.log.Errorf("%s\n", err.Error())
		return TimeData{}, err
	}
	return TimeData{
		hour:   uint32(t.Hour()),
		minute: uint32(t.Minute()),
	}, nil
}

func (j *JobCalendarData) convertStr2NumberWithLimit(str string, numRange NumberRange) (uint32, error) {
	num, err := strconv.Atoi(str)
	if err != nil || uint32(num) < numRange.min || uint32(num) > numRange.max {
		return defaultNum, fmt.Errorf(defaultTimeStrErrorMsg)
	}
	return uint32(num), nil
}

// period: 2:12 - 3:14
// clock: 2:12
func (j *JobCalendarData) parseTimePeriod(period string) (TimeData, TimeData, error) {
	str := strings.ReplaceAll(period, spaceChar, "")
	clocks := strings.Split(str, clockSeparator)
	if len(clocks) > 2 {
		return TimeData{}, TimeData{}, fmt.Errorf(defaultTimePeriodErrorMsg)
	}
	timeFirst, err := j.parseClockStr(clocks[0])
	if err != nil {
		return TimeData{}, TimeData{}, fmt.Errorf(defaultTimePeriodErrorMsg)
	}
	timeSecond, err := j.parseClockStr(clocks[1])
	if err != nil {
		return TimeData{}, TimeData{}, fmt.Errorf(defaultTimePeriodErrorMsg)
	}

	if isTimeDataFirstEarlierThanSecond(timeFirst, timeSecond) {
		return timeFirst, timeSecond, nil
	}

	return timeSecond, timeFirst, nil
}

func (j *JobCalendarData) timeByWeekdayAndOffset(day time.Weekday, clock TimeData) time.Time {
	now := time.Now()
	dayDiffNum := j.weekDayCurrent2Target(now.Weekday(), day)
	return time.Date(now.Year(), now.Month(), now.Day()+dayDiffNum,
		int(clock.hour), int(clock.minute), 0, 0, now.Location())
}

func (j *JobCalendarData) timeOfWeekdayStartFromBeginning(day time.Weekday) FlattenEvents {
	flattenEvents := FlattenEvents{
		start: []time.Time{j.timeByWeekdayAndOffset(day, TimeData{hour: 0, minute: 0})},
		stop:  []time.Time{},
	}
	return flattenEvents
}

func (j *JobCalendarData) sortFlattenEventsFromEarlier2Later() {
	j.log.Debug("sort events")
	events := j.flattenEvents
	sort.Slice(events.start, func(i, j int) bool {
		return events.start[i].Before(events.start[j])
	})
	sort.Slice(events.stop, func(i, j int) bool {
		return events.stop[i].Before(events.stop[j])
	})
}

// TODO: refactor to use for loop, currently no idea how to use code for
// time.Sunday & time.rawData.Sunday
func (j *JobCalendarData) convertWeekScheduleToEvents() {
	j.convertDayScheduleToEvents(time.Sunday, j.rawData.Sunday)
	j.convertDayScheduleToEvents(time.Monday, j.rawData.Monday)
	j.convertDayScheduleToEvents(time.Tuesday, j.rawData.Tuesday)
	j.convertDayScheduleToEvents(time.Wednesday, j.rawData.Wednesday)
	j.convertDayScheduleToEvents(time.Thursday, j.rawData.Thursday)
	j.convertDayScheduleToEvents(time.Friday, j.rawData.Friday)
	j.convertDayScheduleToEvents(time.Saturday, j.rawData.Saturday)
}

func (j *JobCalendarData) convertDayScheduleToEvents(day time.Weekday, clock string) {
	if allDayClockStr == strings.Trim(clock, spaceChar) {
		j.log.Debugf("%s work all day", day.String())
		j.scheduleStartEventWhenDayBegin(day)
		return
	}
	j.scheduleEvents(day, clock)
}

func containsLetter(s string) bool {
	for _, c := range s {
		if unicode.IsLetter(c) {
			return true
		}
	}
	return false
}

func (j *JobCalendarData) isValidPeriod(str string) bool {
	s := strings.Split(str, clockSeparator)
	if len(s) != 2 {
		j.log.Errorf("invalid caledar string %s, contains too many clock string", str)
		return false
	}
	t1 := strings.Trim(s[0], spaceChar)
	t2 := strings.Trim(s[1], spaceChar)
	if t1 == t2 {
		j.log.Errorf("invalid caledar string %s, 2 clock strings equal", str)
		return false
	}

	if containsLetter(t1) || containsLetter(t2) {
		j.log.Errorf("invalid caledar string %s, contains letter", str)
		return false
	}

	return true
}

func (j *JobCalendarData) scheduleEvents(day time.Weekday, clock string) {
	periods := strings.Split(clock, timePeriodSeparator)
	events := make([]SingleEvent, 0)
	flattenEvents := FlattenEvents{
		start: []time.Time{},
		stop:  []time.Time{},
	}

loop:
	for _, period := range periods {
		if !j.isValidPeriod(period) {
			continue loop
		}
		t1, t2, err := j.parseTimePeriod(period)
		if err != nil {
			j.log.Errorf("error parse time period %s, error: %s", period, err)
			continue loop
		}
		events = append(events, SingleEvent{
			start: j.timeByWeekdayAndOffset(day, t1),
			stop:  j.timeByWeekdayAndOffset(day, t2),
		})
		flattenEvents.start = append(
			flattenEvents.start,
			j.timeByWeekdayAndOffset(day, t1),
		)
		flattenEvents.stop = append(
			flattenEvents.stop,
			j.timeByWeekdayAndOffset(day, t2),
		)
	}

	if len(flattenEvents.start) == 0 {
		j.log.Debugf("empty flatten start event, add start event to day start")
		j.scheduleStartEventWhenDayBegin(day)
		return
	}

	j.events[day] = events
	j.flattenEvents.start = append(j.flattenEvents.start, flattenEvents.start...)
	j.flattenEvents.stop = append(j.flattenEvents.stop, flattenEvents.stop...)
}

func (j *JobCalendarData) scheduleStartEventWhenDayBegin(day time.Weekday) {
	flattenEvent := j.timeOfWeekdayStartFromBeginning(day)
	j.flattenEvents.start = append(j.flattenEvents.start, flattenEvent.start[0])
	j.events[day] = []SingleEvent{
		{start: flattenEvent.start[0]},
	}
}

func (j *JobCalendarData) isSameCalendar(newc ConfigCalendar) bool {
	equal := reflect.DeepEqual(j.rawData, newc)
	if !equal {
		j.log.Debug("config changed")
		j.log.Debugf("previous: %+v", j.rawData)
		j.log.Debugf("new: %+v", newc)
	}
	return equal
}

func (j *JobCalendarData) printEvents() {
	weekdays := []time.Weekday{
		time.Sunday,
		time.Monday,
		time.Tuesday,
		time.Wednesday,
		time.Thursday,
		time.Friday,
		time.Saturday,
	}

	for _, d := range weekdays {
		j.log.Debugf("%d start flattenEvents: %+v",
			len(j.flattenEvents.start),
			j.flattenEvents.start,
		)
		j.log.Debugf("%d stop flattenEvents: %+v",
			len(j.flattenEvents.stop),
			j.flattenEvents.stop,
		)
		j.log.Debugf("%d events on %s", len(j.events[d]), d.String())
		for _, e := range j.events[d] {
			if e.stop.IsZero() {
				j.log.Debugf("%s: start: %s",
					d.String(),
					e.start,
				)
			} else {
				j.log.Debugf("%s: start: %s, end: %s",
					d.String(),
					e.start,
					e.stop,
				)
			}
		}
	}
}
