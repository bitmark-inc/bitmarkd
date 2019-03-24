package main

import (
	"fmt"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/bitmark-inc/logger"
)

var channel chan struct{}
var calendarLogging logger.Configuration

var (
	defaultCalendar = ConfigCalendar{
		Monday:    "",
		Tuesday:   "",
		Wednesday: "",
		Thursday:  "",
		Friday:    "",
		Saturday:  "",
		Sunday:    "",
	}
)

func setupTestCalendar() *JobCalendarData {
	setupTestCalendarLogger()
	channel = make(chan struct{})
	now := time.Now()
	j := &JobCalendarData{
		flattenEvents: FlattenEvents{
			start: []time.Time{},
			stop:  []time.Time{},
		},
		events: map[time.Weekday][]SingleEvent{
			time.Sunday:    []SingleEvent{{start: now, stop: now}},
			time.Monday:    []SingleEvent{{start: now, stop: now}},
			time.Tuesday:   []SingleEvent{{start: now, stop: now}},
			time.Wednesday: []SingleEvent{{start: now, stop: now}},
			time.Thursday:  []SingleEvent{{start: now, stop: now}},
			time.Friday:    []SingleEvent{{start: now, stop: now}},
			time.Saturday:  []SingleEvent{{start: now, stop: now}},
		},
		rawData:           ConfigCalendar{},
		rescheduleChannel: channel,
		log:               logger.New("test"),
	}
	return j
}

func setupTestCalendarLogger() {
	_ = os.Mkdir(logDirectory, 0770)
	calendarLogging = loggerConfiguration()
	_ = logger.Initialise(calendarLogging)
}

func teardownCalendar() {
	logger.Finalise()
	removeTestFiles()
}

func stringifyTime(t time.Time) string {
	return t.Format("Jan 2 15:04")
}

func TestIsSameTime(t *testing.T) {
	fixture := []struct {
		t1       TimeData
		t2       TimeData
		expected bool
	}{
		{
			TimeData{
				hour:   1,
				minute: 2,
			},
			TimeData{
				hour:   1,
				minute: 2,
			},
			true,
		},
		{
			TimeData{
				hour:   1,
				minute: 3,
			},
			TimeData{
				hour:   3,
				minute: 4,
			},
			false,
		},
		{
			TimeData{
				hour:   1,
				minute: 3,
			},
			TimeData{
				hour:   1,
				minute: 4,
			},
			false,
		},
		{
			TimeData{
				hour:   1,
				minute: 3,
			},
			TimeData{
				hour:   2,
				minute: 3,
			},
			false,
		},
	}

	for i, s := range fixture {
		actual := isSameTime(s.t1, s.t2)
		if actual != s.expected {
			t.Errorf("%dth decide same time error", i)
			t.Errorf("given input t1 %v and t2 %v", s.t1, s.t2)
			t.Errorf("expect %t but get %t", s.expected, actual)
		}
	}
}

func TestWeekDayCurrent2Target(t *testing.T) {
	j := &JobCalendarData{}
	expects := []struct {
		current  time.Weekday
		target   time.Weekday
		expected int
	}{
		{time.Monday, time.Friday, 4},
		{time.Friday, time.Tuesday, -3},
		{time.Wednesday, time.Wednesday, 0},
	}

	for _, s := range expects {
		actual := j.weekDayCurrent2Target(s.current, s.target)
		if s.expected != actual {
			t.Errorf("error calculate day distance, expected %s to %s is %d day but get %d",
				s.current, s.target, s.expected, actual)
		}
	}
}

func TestConvertStr2NumberWithLimit(t *testing.T) {
	j := &JobCalendarData{}
	timeErr := fmt.Errorf(defaultTimeStrErrorMsg)
	expects := []struct {
		str      string
		r        NumberRange
		expected uint32
		err      error
	}{
		{"01", NumberRange{max: 5, min: 0}, 1, nil},
		{"13", NumberRange{max: 60, min: 10}, 13, nil},
		{"-1", NumberRange{max: 100, min: 0}, defaultNum, timeErr},
		{"25", NumberRange{max: 24, min: 12}, defaultNum, timeErr},
	}

	for _, s := range expects {
		num, err := j.convertStr2NumberWithLimit(s.str, s.r)
		if (err != nil && s.err.Error() != err.Error()) || (err == nil && s.err != err) {
			t.Errorf("error convert string %s to hour, expect error %s but get %s",
				s.str, s.err, err)
		}
		if s.expected != num {
			t.Errorf("error convert string %s to hour, expect %d but get %d",
				s.str, s.expected, num)
		}
	}
}

func TestParseClockStr(t *testing.T) {
	j := setupTestCalendar()
	defer teardown()

	e := "Error"
	expects := []struct {
		clockStr string
		expected TimeData
		err      interface{}
	}{
		{"-2:05", TimeData{hour: 0, minute: 0}, e},
		{"5:-3", TimeData{hour: 0, minute: 0}, e},
		{"25:05", TimeData{hour: 0, minute: 0}, e},
		{"14:78", TimeData{hour: 0, minute: 0}, e},
		{"24:12", TimeData{hour: 0, minute: 0}, e},
		{"01:23", TimeData{hour: 1, minute: 23}, nil},
		{"23:47", TimeData{hour: 23, minute: 47}, nil},
		{"24:00", TimeData{hour: 24, minute: 0}, nil},
	}

	for _, s := range expects {
		_, err := j.parseClockStr(s.clockStr)
		if err != nil && s.err == nil {
			t.Errorf("error convert string %s to hour, expect error %s but get %s",
				s.clockStr, s.err, err)
		}
	}
}

func TestTimeByWeekdayAndOffset(t *testing.T) {
	j := &JobCalendarData{}
	now := time.Now()

	sunday := time.Date(
		now.Year(),
		now.Month(),
		now.Day()+int(time.Sunday)-int(now.Weekday()),
		7, 12, 0, 0,
		now.Location(),
	)
	friday := time.Date(
		now.Year(),
		now.Month(),
		now.Day()+int(time.Friday)-int(now.Weekday()),
		18, 0, 0, 0,
		now.Location(),
	)

	expects := []struct {
		day      time.Weekday
		clock    TimeData
		expected time.Time
	}{
		{time.Sunday, TimeData{hour: 7, minute: 12}, sunday},
		{time.Friday, TimeData{hour: 18, minute: 0}, friday},
	}

	for _, s := range expects {
		actual := j.timeByWeekdayAndOffset(s.day, s.clock)
		if actual != s.expected {
			t.Errorf("error convert week day into time, expect %v but get %v",
				s.expected, actual)
		}
	}
}

func TestDayStartFromBeginning(t *testing.T) {
	j := &JobCalendarData{}
	now := time.Now()
	e := struct {
		weekday  time.Weekday
		clock    string
		expected FlattenEvents
	}{
		now.Weekday(), "",
		FlattenEvents{
			start: []time.Time{time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())},
			stop:  []time.Time{},
		},
	}

	actual := j.timeOfWeekdayStartFromBeginning(e.weekday)
	if len(actual.start) != len(e.expected.start) || actual.start[0] != e.expected.start[0] {
		t.Errorf("error get time starting from day, expect %v but get %v",
			e.expected, actual)
	}
}

func TestScheduleStartEventWhenDayBegin(t *testing.T) {
	j := setupTestCalendar()
	defer teardownCalendar()

	now := time.Now()
	expects := struct {
		weekday  time.Weekday
		clock    string
		expected FlattenEvents
	}{
		now.Weekday(),
		"",
		FlattenEvents{
			start: []time.Time{time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())},
			stop:  []time.Time{},
		},
	}

	j.scheduleStartEventWhenDayBegin(expects.weekday)

	if len(j.flattenEvents.start) != len(expects.expected.start) ||
		len(j.flattenEvents.stop) != len(expects.expected.stop) ||
		!isTimeSliceEqual(j.flattenEvents.start, expects.expected.start) ||
		!isTimeSliceEqual(j.flattenEvents.stop, expects.expected.stop) {
		t.Errorf("error gettings time slice, expect %v but get %v",
			expects.expected, j.flattenEvents)
	}
}

func TestScheduleEvents(t *testing.T) {
	j := setupTestCalendar()
	defer teardownCalendar()

	now := time.Now()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	expects := []struct {
		weekday         time.Weekday
		clock           string
		expectedFlatten FlattenEvents
		expectedEvents  []SingleEvent
	}{
		{
			now.Weekday(),
			"1:23-4:56, 12:33-18:00",
			FlattenEvents{
				start: []time.Time{
					time.Date(now.Year(), now.Month(), now.Day(), 1, 23, 0, 0, now.Location()),
					time.Date(now.Year(), now.Month(), now.Day(), 12, 33, 0, 0, now.Location()),
				},
				stop: []time.Time{
					time.Date(now.Year(), now.Month(), now.Day(), 4, 56, 0, 0, now.Location()),
					time.Date(now.Year(), now.Month(), now.Day(), 18, 00, 0, 0, now.Location()),
				},
			},
			[]SingleEvent{
				{
					start: time.Date(now.Year(), now.Month(), now.Day(), 1, 23, 0, 0, now.Location()),
					stop:  time.Date(now.Year(), now.Month(), now.Day(), 4, 56, 0, 0, now.Location()),
				},
				{
					start: time.Date(now.Year(), now.Month(), now.Day(), 12, 33, 0, 0, now.Location()),
					stop:  time.Date(now.Year(), now.Month(), now.Day(), 18, 00, 0, 0, now.Location()),
				},
			},
		},
		{
			now.Weekday(),
			"1:00-2:00, 2:00-3:00, 3:00-4:00",
			FlattenEvents{
				start: []time.Time{
					dayStart.Add(time.Duration(1) * time.Hour),
					dayStart.Add(time.Duration(2) * time.Hour),
					dayStart.Add(time.Duration(3) * time.Hour),
				},
				stop: []time.Time{
					dayStart.Add(time.Duration(2) * time.Hour),
					dayStart.Add(time.Duration(3) * time.Hour),
					dayStart.Add(time.Duration(4) * time.Hour),
				},
			},
			[]SingleEvent{
				{
					start: dayStart.Add(time.Duration(1) * time.Hour),
					stop:  dayStart.Add(time.Duration(2) * time.Hour),
				},
				{
					start: dayStart.Add(time.Duration(2) * time.Hour),
					stop:  dayStart.Add(time.Duration(3) * time.Hour),
				},
				{
					start: dayStart.Add(time.Duration(3) * time.Hour),
					stop:  dayStart.Add(time.Duration(4) * time.Hour),
				},
			},
		},
	}

	for i, s := range expects {
		j.resetEvents()
		j.scheduleEvents(s.weekday, s.clock)
		if len(j.flattenEvents.start) != len(s.expectedFlatten.start) ||
			len(j.flattenEvents.stop) != len(s.expectedFlatten.stop) ||
			!isTimeSliceEqual(j.flattenEvents.start, s.expectedFlatten.start) ||
			!isTimeSliceEqual(j.flattenEvents.stop, s.expectedFlatten.stop) {
			t.Errorf("%dth test fail, error getting time slice", i)
			for _, e := range s.expectedFlatten.start {
				t.Errorf("expect flatten start %v",
					stringifyTime(e))
			}
			for _, e := range s.expectedFlatten.stop {
				t.Errorf("expect flatten stop %v", stringifyTime(e))
			}
			for _, e := range j.flattenEvents.start {
				t.Errorf("actual flatten start %v", stringifyTime(e))
			}
			for _, e := range j.flattenEvents.stop {
				t.Errorf("actual flatten stop %v", stringifyTime(e))
			}
		}
		if len(j.events[s.weekday]) != len(s.expectedEvents) {
			t.Errorf("%dth test fail, error getting time slice", i)
			for _, e := range s.expectedEvents {
				t.Errorf("expect events start %s, end %s",
					stringifyTime(e.start),
					stringifyTime(e.stop),
				)
			}
			for _, e := range j.events[s.weekday] {
				t.Errorf("actual events start %s, end %s",
					stringifyTime(e.start),
					stringifyTime(e.stop),
				)
			}
		}
	}
}

func TestIsSameCalendar(t *testing.T) {
	j := setupTestCalendar()
	defer teardownCalendar()

	expects := []struct {
		orig     ConfigCalendar
		new      ConfigCalendar
		expected bool
	}{
		{
			defaultCalendar,
			defaultCalendar,
			true,
		},
		{
			defaultCalendar,
			ConfigCalendar{
				Monday:    "",
				Tuesday:   "1:00-2:00",
				Wednesday: "",
				Thursday:  "",
				Friday:    "",
				Saturday:  "",
				Sunday:    "",
			},
			false,
		},
	}

	for _, s := range expects {
		j.setNewCalendar(s.orig)
		actual := j.isSameCalendar(s.new)
		if actual != s.expected {
			t.Errorf("error checking config, expect compare result of %v to %v is %t",
				s.orig, s.new, s.expected)
		}
	}
}

func TestSortFlattenEventsFromEarlier2Later(t *testing.T) {
	j := setupTestCalendar()
	defer teardownCalendar()
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayEnd := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 0, 0, now.Location())
	j.flattenEvents.start = []time.Time{now, todayEnd, todayStart}
	j.flattenEvents.stop = []time.Time{todayEnd, todayStart, now}

	expected := &JobCalendarData{
		flattenEvents: FlattenEvents{
			start: []time.Time{
				todayStart, now, todayEnd,
			},
			stop: []time.Time{
				todayStart, now, todayEnd,
			},
		},
	}

	j.sortFlattenEventsFromEarlier2Later()

	if !reflect.DeepEqual(
		j.flattenEvents.start,
		expected.flattenEvents.start) ||
		!reflect.DeepEqual(
			j.flattenEvents.stop,
			expected.flattenEvents.stop) {
		t.Errorf("error sorting events, expected %v but get %v",
			expected.flattenEvents, j.flattenEvents)
	}
}

func afterMinuteFromBase(base time.Time, minute int) time.Time {
	return base.Add(time.Duration(minute) * time.Minute)
}

func nextWeekFromBase(base time.Time) time.Time {
	return base.Add(time.Duration(24*7) * time.Hour)
}

func TestIsEventAlreadyExist(t *testing.T) {
	now := time.Now()
	fixture := []struct {
		times    []time.Time
		event    time.Time
		expected bool
	}{
		{[]time.Time{
			now,
			afterMinuteFromBase(now, 1),
			afterMinuteFromBase(now, 2),
		}, now, true},
		{[]time.Time{
			now,
			afterMinuteFromBase(now, 1),
			afterMinuteFromBase(now, 2),
		}, afterMinuteFromBase(now, 3), false},
	}

	for _, s := range fixture {
		actual, _ := isEventAlreadyExist(s.times, s.event)
		if actual != s.expected {
			t.Errorf("error check time exist, expected %s in %v is %t but get %t",
				s.event, s.times, s.expected, actual)
		}
	}
}

func isTimeSliceEqual(a, b []time.Time) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if !v.Equal(b[i]) {
			return false
		}
	}
	return true
}

func TestRescheduleStartEventsPrior(t *testing.T) {
	now := time.Now()
	nextWeek := nextWeekFromBase(now)
	j := &JobCalendarData{}
	oneMinuteAfter := afterMinuteFromBase(now, 1)
	fiveMinuteAfter := afterMinuteFromBase(now, 5)
	oneMinuteBefore := afterMinuteFromBase(now, -1)
	fiveMinuteBefore := afterMinuteFromBase(now, -5)
	nextWeekOneMinuteBefore := oneMinuteBefore.AddDate(0, 0, 7)
	nextWeekFiveMinuteBefore := fiveMinuteBefore.AddDate(0, 0, 7)
	fixture := []struct {
		times    []time.Time
		event    time.Time
		expected []time.Time
	}{
		{
			[]time.Time{now, oneMinuteAfter, fiveMinuteAfter},
			now,
			[]time.Time{oneMinuteAfter, fiveMinuteAfter, nextWeek},
		},
		{
			[]time.Time{oneMinuteAfter, fiveMinuteAfter},
			now,
			[]time.Time{oneMinuteAfter, fiveMinuteAfter},
		},
		{
			[]time.Time{oneMinuteBefore, fiveMinuteBefore},
			now,
			[]time.Time{nextWeekOneMinuteBefore, nextWeekFiveMinuteBefore},
		},
		{
			[]time.Time{oneMinuteBefore, now, fiveMinuteAfter},
			now,
			[]time.Time{fiveMinuteAfter, nextWeekOneMinuteBefore, nextWeek},
		},
	}

	for i, s := range fixture {
		j.flattenEvents.stop = s.times
		j.rescheduleStopEventsPrior(s.event)
		if !isTimeSliceEqual(s.expected, j.flattenEvents.stop) {
			t.Errorf("%dth test fail, schedule next week event, now is %s",
				i, stringifyTime(now))
			for _, e := range s.expected {
				t.Errorf("expected time: %s", stringifyTime(e))
			}
			for _, e := range j.flattenEvents.stop {
				t.Errorf("actual time: %s", stringifyTime(e))
			}
		}
	}
}

func TestRescheduleStopEventsPrior(t *testing.T) {
	now := time.Now()
	nextWeek := nextWeekFromBase(now)
	j := &JobCalendarData{}
	oneMinuteAfter := afterMinuteFromBase(now, 1)
	fiveMinuteAfter := afterMinuteFromBase(now, 5)
	oneMinuteBefore := afterMinuteFromBase(now, -1)
	fiveMinuteBefore := afterMinuteFromBase(now, -5)
	nextWeekOneMinuteBefore := oneMinuteBefore.AddDate(0, 0, 7)
	nextWeekFiveMinuteBefore := fiveMinuteBefore.AddDate(0, 0, 7)
	fixture := []struct {
		times    []time.Time
		event    time.Time
		expected []time.Time
	}{
		{
			[]time.Time{now, oneMinuteAfter, fiveMinuteAfter},
			now,
			[]time.Time{oneMinuteAfter, fiveMinuteAfter, nextWeek},
		},
		{
			[]time.Time{oneMinuteAfter, fiveMinuteAfter},
			now,
			[]time.Time{oneMinuteAfter, fiveMinuteAfter},
		},
		{
			[]time.Time{oneMinuteBefore, fiveMinuteBefore},
			now,
			[]time.Time{nextWeekOneMinuteBefore, nextWeekFiveMinuteBefore},
		},
		{
			[]time.Time{oneMinuteBefore, now, fiveMinuteAfter},
			now,
			[]time.Time{fiveMinuteAfter, nextWeekOneMinuteBefore, nextWeek},
		},
	}

	for i, s := range fixture {
		j.flattenEvents.stop = s.times
		j.rescheduleStopEventsPrior(s.event)
		if !isTimeSliceEqual(s.expected, j.flattenEvents.stop) {
			t.Errorf("%dth test fail, schedule next week event, now is %s",
				i, stringifyTime(now))
			for _, e := range s.expected {
				t.Errorf("expected time: %s", stringifyTime(e))
			}
			for _, e := range j.flattenEvents.stop {
				t.Errorf("actual time: %s", stringifyTime(e))
			}
		}
	}
}

func TestRemoveEventFrom(t *testing.T) {
	now := time.Now()
	j := &JobCalendarData{}
	fixture := []struct {
		times    []time.Time
		event    time.Time
		expected []time.Time
	}{
		{
			[]time.Time{now, afterMinuteFromBase(now, 1)}, now,
			[]time.Time{afterMinuteFromBase(now, 1)},
		},
		{
			[]time.Time{now, afterMinuteFromBase(now, 1)}, afterMinuteFromBase(now, 2),
			[]time.Time{now, afterMinuteFromBase(now, 1)},
		},
	}
	for _, s := range fixture {
		actual, _ := j.removeEventFrom(s.times, s.event)
		if !isTimeSliceEqual(s.expected, actual) {
			t.Errorf("error schedule next week event, expect %v but get %v",
				s.expected, actual)
		}
	}
}

func TestIsTimeBooked(t *testing.T) {
	now := time.Now()
	weekDay := now.Weekday()
	j := setupTestCalendar()
	defer teardownCalendar()

	j.events[weekDay] = []SingleEvent{
		{
			start: afterMinuteFromBase(now, 1),
			stop:  afterMinuteFromBase(now, 3),
		},
	}

	fixture := []struct {
		target   time.Time
		expected bool
	}{
		{now, false},
		{afterMinuteFromBase(now, 1), true},
		{afterMinuteFromBase(now, 4), false},
	}

	for i, s := range fixture {
		actual := j.isTimeBooked(s.target)
		if actual != s.expected {
			t.Errorf("%dth error checking time, %s in range of %s ~ %s, expect %t but get %t",
				i, stringifyTime(s.target), stringifyTime(j.events[weekDay][0].start), stringifyTime(j.events[weekDay][0].stop),
				s.expected, actual)
		}
	}

	j.events[weekDay] = []SingleEvent{
		{
			start: afterMinuteFromBase(now, 1),
		},
	}

	fixture = []struct {
		target   time.Time
		expected bool
	}{
		{now, false},
		{afterMinuteFromBase(now, 1), true},
		{afterMinuteFromBase(now, 4), true},
	}

	for i, s := range fixture {
		actual := j.isTimeBooked(s.target)
		if actual != s.expected {
			t.Errorf("%dth error checking time, %s in range of %s ~ %s, expect %t but get %t",
				i, stringifyTime(s.target), stringifyTime(j.events[weekDay][0].start), stringifyTime(j.events[weekDay][0].stop),
				s.expected, actual)
		}
	}
}

func TestNotifyJobManager(t *testing.T) {
	j := setupTestCalendar()
	defer teardownCalendar()

	received := false
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		<-channel
		received = true
		wg.Done()
	}()

	j.notifyJobManager()
	wg.Wait()

	if received != true {
		t.Errorf("error job manager didn't receive reschedule event.")
	}
}

func TestPickNextStartEvent(t *testing.T) {
	j := setupTestCalendar()
	defer teardownCalendar()

	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	tomorrow := now.AddDate(0, 0, 1)
	fixture := []struct {
		weekday       time.Weekday
		flattenEvents FlattenEvents
		expected      interface{}
	}{
		{
			now.Weekday(),
			FlattenEvents{
				start: []time.Time{
					yesterday,
					now,
					tomorrow,
				},
				stop: []time.Time{},
			},
			tomorrow,
		},
		{
			now.Weekday(),
			FlattenEvents{
				start: []time.Time{
					yesterday,
					now,
				},
				stop: []time.Time{},
			},
			nil,
		},
	}

	for i, s := range fixture {
		j.flattenEvents = s.flattenEvents
		actual := j.pickNextStartEvent(now)
		if actual != s.expected {
			t.Errorf("%dth test fail cannot get correct next start events", i)
			t.Errorf("now: %s, yesterday: %s, tomorrow: %s",
				stringifyTime(now), stringifyTime(yesterday), stringifyTime(tomorrow))
			for _, start := range j.flattenEvents.start {
				t.Errorf("flatten events of start: %s", start)
			}
			for _, stop := range j.flattenEvents.stop {
				t.Errorf("flatten events of stop: %s", stop)
			}

			t.Errorf("expect next event at %s but get %s",
				stringifyTime(s.expected.(time.Time)),
				stringifyTime(actual.(time.Time)),
			)
		}
	}
}

func TestPickNextStopEvent(t *testing.T) {
	j := setupTestCalendar()
	defer teardownCalendar()

	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	tomorrow := now.AddDate(0, 0, 1)
	fixture := []struct {
		weekday       time.Weekday
		flattenEvents FlattenEvents
		expected      interface{}
	}{
		{
			now.Weekday(),
			FlattenEvents{
				start: []time.Time{},
				stop: []time.Time{
					now,
					tomorrow,
				},
			},
			tomorrow,
		},
		{
			now.Weekday(),
			FlattenEvents{
				start: []time.Time{},
				stop: []time.Time{
					yesterday,
					now,
				},
			},
			nil,
		},
	}

	for i, s := range fixture {
		j.flattenEvents = s.flattenEvents
		actual := j.pickNextStopEvent(now)
		if actual != s.expected {
			t.Errorf("%dth test fail cannot get correct next stop event", i)
			t.Errorf("now: %s, yesterday: %s, tomorrow: %s",
				stringifyTime(now), stringifyTime(yesterday), stringifyTime(tomorrow))
			for _, start := range j.flattenEvents.start {
				t.Errorf("flatten events of start: %s", start)
			}
			for _, stop := range j.flattenEvents.stop {
				t.Errorf("flatten events of stop: %s", stop)
			}

			t.Errorf("expect next event at %s but get %s",
				stringifyTime(s.expected.(time.Time)),
				stringifyTime(actual.(time.Time)),
			)
		}
	}
}

func TestPickInitialiseStartEvent(t *testing.T) {
	j := setupTestCalendar()
	defer teardownCalendar()

	now := time.Now()
	oneHourBefore := time.Date(now.Year(), now.Month(), now.Day(), now.Hour()-1,
		0, 0, 0, now.Location())
	fiveMinuteAfter := now.Add(time.Duration(5) * time.Minute)
	tenMinuteAfter := now.Add(time.Duration(10) * time.Minute)
	tomorrow := now.AddDate(0, 0, 1)
	tomorrowOneHourBefore := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(),
		tomorrow.Hour()-1, 0, 0, 0, tomorrow.Location())
	tomorrowFiveMinuteAfter := tomorrow.Add(time.Duration(5) * time.Minute)
	fixture := []struct {
		weekday       time.Weekday
		flattenEvents FlattenEvents
		events        map[time.Weekday][]SingleEvent
		expected      interface{}
	}{
		{
			now.Weekday(),
			FlattenEvents{
				start: []time.Time{
					oneHourBefore,
					tomorrowOneHourBefore,
				},
				stop: []time.Time{
					fiveMinuteAfter,
					tomorrowFiveMinuteAfter,
				},
			},
			map[time.Weekday][]SingleEvent{
				time.Monday: []SingleEvent{
					{
						start: oneHourBefore.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Monday)),
						stop: fiveMinuteAfter.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Monday)),
					},
				},
				time.Tuesday: []SingleEvent{
					{
						start: oneHourBefore.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Tuesday)),
						stop: fiveMinuteAfter.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Tuesday)),
					},
				},
				time.Wednesday: []SingleEvent{
					{
						start: oneHourBefore.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Wednesday)),
						stop: fiveMinuteAfter.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Wednesday)),
					},
				},
				time.Thursday: []SingleEvent{
					{
						start: oneHourBefore.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Thursday)),
						stop: fiveMinuteAfter.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Thursday)),
					},
				},
				time.Friday: []SingleEvent{
					{
						start: oneHourBefore.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Friday)),
						stop: fiveMinuteAfter.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Friday)),
					},
				},
				time.Saturday: []SingleEvent{
					{
						start: oneHourBefore.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Saturday)),
						stop: fiveMinuteAfter.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Saturday)),
					},
				},
				time.Sunday: []SingleEvent{
					{
						start: oneHourBefore.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Sunday)),
						stop: fiveMinuteAfter.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Sunday)),
					},
				},
			},
			now.Add(time.Duration(5) * time.Second),
		},
		{
			now.Weekday(),
			FlattenEvents{
				start: []time.Time{
					fiveMinuteAfter,
				},
				stop: []time.Time{
					tenMinuteAfter,
				},
			},
			map[time.Weekday][]SingleEvent{
				time.Monday: []SingleEvent{
					{
						start: fiveMinuteAfter.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Monday)),
						stop: tenMinuteAfter.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Monday)),
					},
				},
				time.Tuesday: []SingleEvent{
					{
						start: fiveMinuteAfter.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Monday)),
						stop: tenMinuteAfter.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Monday)),
					},
				},
				time.Wednesday: []SingleEvent{
					{
						start: fiveMinuteAfter.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Monday)),
						stop: tenMinuteAfter.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Monday)),
					},
				},
				time.Thursday: []SingleEvent{
					{
						start: fiveMinuteAfter.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Monday)),
						stop: tenMinuteAfter.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Monday)),
					},
				},
				time.Friday: []SingleEvent{
					{
						start: fiveMinuteAfter.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Monday)),
						stop: tenMinuteAfter.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Monday)),
					},
				},
				time.Saturday: []SingleEvent{
					{
						start: fiveMinuteAfter.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Monday)),
						stop: tenMinuteAfter.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Monday)),
					},
				},
				time.Sunday: []SingleEvent{
					{
						start: fiveMinuteAfter.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Monday)),
						stop: tenMinuteAfter.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Monday)),
					},
				},
			},
			fiveMinuteAfter,
		},
	}

	for i, s := range fixture {
		j.flattenEvents = s.flattenEvents
		j.events = s.events
		actual := j.pickInitialiseStartEvent(now)
		if actual != s.expected {
			t.Errorf("%dth test fail, cannot get correct next events", i)
			t.Errorf("now: %s, 1 hour before: %s, 5 min after: %s, 10 min after: %s",
				stringifyTime(now),
				stringifyTime(oneHourBefore),
				stringifyTime(fiveMinuteAfter),
				stringifyTime(tenMinuteAfter),
			)
			for _, s := range j.events[s.weekday] {
				t.Errorf("time period: %s - %s", s.start, s.stop)
			}
			t.Errorf("expect next event at %s but get %s",
				stringifyTime(s.expected.(time.Time)),
				stringifyTime(actual.(time.Time)),
			)
		}
	}
}

func TestPickInitialiseStopEvent(t *testing.T) {
	j := setupTestCalendar()
	defer teardownCalendar()

	now := time.Now()
	oneHourBefore := time.Date(now.Year(), now.Month(), now.Day(), now.Hour()-1,
		0, 0, 0, now.Location())
	fiveMinuteBefore := now.Add(-time.Duration(5) * time.Minute)
	tenMinuteAfter := now.Add(time.Duration(10) * time.Minute)
	tomorrow := now.AddDate(0, 0, 1)
	tomorrowOneHourBefore := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(),
		tomorrow.Hour()-1, 0, 0, 0, tomorrow.Location())
	tomorrowFiveMinuteBefore := tomorrow.Add(-time.Duration(5) * time.Minute)
	fixture := []struct {
		weekday       time.Weekday
		flattenEvents FlattenEvents
		events        map[time.Weekday][]SingleEvent
		expected      interface{}
	}{
		{
			now.Weekday(),
			FlattenEvents{
				start: []time.Time{
					oneHourBefore,
					tomorrowOneHourBefore,
				},
				stop: []time.Time{
					fiveMinuteBefore,
					tomorrowFiveMinuteBefore,
				},
			},
			map[time.Weekday][]SingleEvent{
				time.Monday: []SingleEvent{
					{
						start: oneHourBefore.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Monday)),
						stop: fiveMinuteBefore.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Monday)),
					},
				},
				time.Tuesday: []SingleEvent{
					{
						start: oneHourBefore.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Tuesday)),
						stop: fiveMinuteBefore.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Tuesday)),
					},
				},
				time.Wednesday: []SingleEvent{
					{
						start: oneHourBefore.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Wednesday)),
						stop: fiveMinuteBefore.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Wednesday)),
					},
				},
				time.Thursday: []SingleEvent{
					{
						start: oneHourBefore.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Thursday)),
						stop: fiveMinuteBefore.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Thursday)),
					},
				},
				time.Friday: []SingleEvent{
					{
						start: oneHourBefore.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Friday)),
						stop: fiveMinuteBefore.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Friday)),
					},
				},
				time.Saturday: []SingleEvent{
					{
						start: oneHourBefore.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Saturday)),
						stop: fiveMinuteBefore.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Saturday)),
					},
				},
				time.Sunday: []SingleEvent{
					{
						start: oneHourBefore.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Sunday)),
						stop: fiveMinuteBefore.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Sunday)),
					},
				},
			},
			now.Add(time.Duration(5) * time.Second),
		},
		{
			now.Weekday(),
			FlattenEvents{
				start: []time.Time{
					oneHourBefore,
				},
				stop: []time.Time{
					tenMinuteAfter,
				},
			},
			map[time.Weekday][]SingleEvent{
				time.Monday: []SingleEvent{
					{
						stop: tenMinuteAfter.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Monday)),
						start: oneHourBefore.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Monday)),
					},
				},
				time.Tuesday: []SingleEvent{
					{
						stop: tenMinuteAfter.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Tuesday)),
						start: oneHourBefore.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Tuesday)),
					},
				},
				time.Wednesday: []SingleEvent{
					{
						stop: tenMinuteAfter.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Wednesday)),
						start: oneHourBefore.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Wednesday)),
					},
				},
				time.Thursday: []SingleEvent{
					{
						stop: tenMinuteAfter.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Thursday)),
						start: oneHourBefore.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Thursday)),
					},
				},
				time.Friday: []SingleEvent{
					{
						stop: tenMinuteAfter.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Friday)),
						start: oneHourBefore.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Friday)),
					},
				},
				time.Saturday: []SingleEvent{
					{
						stop: tenMinuteAfter.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Saturday)),
						start: oneHourBefore.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Saturday)),
					},
				},
				time.Sunday: []SingleEvent{
					{
						stop: tenMinuteAfter.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Sunday)),
						start: oneHourBefore.AddDate(0, 0,
							j.weekDayCurrent2Target(now.Weekday(), time.Sunday)),
					},
				},
			},
			tenMinuteAfter,
		},
	}

	for i, s := range fixture {
		j.flattenEvents = s.flattenEvents
		j.events = s.events
		actual := j.pickInitialiseStopEvent(now)
		if actual != s.expected {
			t.Errorf("%dth test fail, cannot get correct next events", i)
			t.Errorf("now: %s, 1 hour before: %s, 5 min before: %s, 10 min before: %s",
				stringifyTime(now),
				stringifyTime(oneHourBefore),
				stringifyTime(fiveMinuteBefore),
				stringifyTime(tenMinuteAfter),
			)
			for _, s := range j.events[s.weekday] {
				t.Errorf("time period: %s - %s", s.start, s.stop)
			}
			t.Errorf("expect next event at %s but get %s",
				stringifyTime(s.expected.(time.Time)),
				stringifyTime(actual.(time.Time)),
			)
			t.Errorf("time period: %s - %s",
				stringifyTime(j.flattenEvents.start[0]),
				stringifyTime(j.flattenEvents.stop[0]),
			)
			t.Errorf("expect next stop time to be %s but get %s",
				stringifyTime(s.expected.(time.Time)), stringifyTime(actual.(time.Time)),
			)
		}
	}
}

func TestIsValidPeriod(t *testing.T) {
	j := setupTestCalendar()
	defer teardownCalendar()
	fixture := []struct {
		period   string
		expected bool
	}{
		{" 1:00 - 1:00 ", false},
		{"2:23 - 4:56 ", true},
		{"1:11 - 2:22 - 3:33", false},
		{"", false},
		{"1:11a-2:34", false},
		{"a:aa-b:bb", false},
		{"1:ab=2:cd", false},
	}

	for _, s := range fixture {
		actual := j.isValidPeriod(s.period)
		if actual != s.expected {
			t.Errorf("wrong period comparison, %s exepect to be %t but get %t",
				s.period, s.expected, actual)
		}
	}
}

func TestIsTimeDataFirstEarlierThanSecond(t *testing.T) {
	fixture := []struct {
		first    TimeData
		second   TimeData
		expected bool
	}{
		{TimeData{hour: 2, minute: 3}, TimeData{hour: 3, minute: 3}, true},
		{TimeData{hour: 2, minute: 3}, TimeData{hour: 1, minute: 3}, false},
		{TimeData{hour: 2, minute: 3}, TimeData{hour: 2, minute: 4}, true},
		{TimeData{hour: 2, minute: 3}, TimeData{hour: 2, minute: 2}, false},
	}

	for i, s := range fixture {
		actual := isTimeDataFirstEarlierThanSecond(s.first, s.second)
		if actual != s.expected {
			t.Errorf("%dth error compare time data", i)
			t.Errorf("first time %v, second time %v, expected %t but get %t",
				s.first, s.second, s.expected, actual)
		}
	}
}

func TestContainsLetter(t *testing.T) {
	fixture := []struct {
		str      string
		expected bool
	}{
		{"123abc", true},
		{"=-`.,/", false},
		{"123098   ", false},
	}

	for _, s := range fixture {
		actual := containsLetter(s.str)
		if actual != s.expected {
			t.Errorf("error checking letter, %s expected %t but get %t", s.str, s.expected, actual)
		}
	}
}

func TestRemoveRedundantStopEvent(t *testing.T) {
	j := setupTestCalendar()
	defer teardownCalendar()

	now := time.Now()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	fixture := []struct {
		flatten  FlattenEvents
		expected FlattenEvents
	}{
		{
			FlattenEvents{
				start: []time.Time{
					dayStart.Add(time.Duration(1) * time.Hour),
					dayStart.Add(time.Duration(2) * time.Hour),
					dayStart.Add(time.Duration(3) * time.Hour),
				},
				stop: []time.Time{
					dayStart.Add(time.Duration(2) * time.Hour),
					dayStart.Add(time.Duration(3) * time.Hour),
					dayStart.Add(time.Duration(4) * time.Hour),
				},
			},
			FlattenEvents{
				start: []time.Time{
					dayStart.Add(time.Duration(1) * time.Hour),
					dayStart.Add(time.Duration(2) * time.Hour),
					dayStart.Add(time.Duration(3) * time.Hour),
				},
				stop: []time.Time{
					dayStart.Add(time.Duration(4) * time.Hour),
				},
			},
		},
	}

	for _, s := range fixture {
		j.flattenEvents = s.flatten
		j.removeRedundantStopEvent()
		if !isTimeSliceEqual(s.expected.start, j.flattenEvents.start) {
			for _, e := range s.expected.start {
				t.Errorf("expect flatten start %s",
					stringifyTime(e))
			}
			for _, e := range j.flattenEvents.start {
				t.Errorf("actual flatten start %s",
					stringifyTime(e))
			}
		}
		if !isTimeSliceEqual(s.expected.stop, j.flattenEvents.stop) {
			for _, e := range s.expected.stop {
				t.Errorf("expect flatten stop %s",
					stringifyTime(e))
			}
			for _, e := range j.flattenEvents.stop {
				t.Errorf("actual flatten stop %s",
					stringifyTime(e))
			}
		}
	}
}

func TestRunForever(t *testing.T) {
	j := setupTestCalendar()
	defer teardownCalendar()

	now := time.Now()
	fixture := []struct {
		flattenEvents FlattenEvents
		expected      bool
	}{
		{
			FlattenEvents{
				start: []time.Time{now},
				stop:  []time.Time{},
			},
			true,
		},
		{
			FlattenEvents{
				start: []time.Time{now},
				stop:  []time.Time{now},
			},
			false,
		},
	}

	for i, s := range fixture {
		j.flattenEvents = s.flattenEvents
		actual := j.runForever()
		if actual != s.expected {
			t.Errorf("%d the test fail, expect %t but get %t", i, s.expected, actual)
		}
	}
}
