package logfeller

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	oneDay = 24 * time.Hour
	// approxOneMonth is the approximate number of days in a month, this is
	// mainly used for comparing offsets
	// If you want to get the correct number of days in a month, use daysIn
	// instead
	approxOneMonth = 30 * oneDay
	oneYear        = 365 * oneDay
)

// WhenRotate helps reason about logic related to rotation of the file.
type WhenRotate string

const (
	Hour  WhenRotate = "h"
	Day   WhenRotate = "d"
	Month WhenRotate = "m"
	Year  WhenRotate = "y"
)

var (
	hourOffsetRegex  = regexp.MustCompile(`^(?P<minutes>\d{2})(?P<seconds>\d{2})$`)
	dayOffsetRegex   = regexp.MustCompile(`^(?P<hours>\d{2})(?P<minutes>\d{2})(?P<seconds>\d{2})$`)
	monthOffsetRegex = regexp.MustCompile(`^(?P<days>\d{2}) (?P<hours>\d{2})(?P<minutes>\d{2})(?P<seconds>\d{2})$`)
	yearOffsetRegex  = regexp.MustCompile(`^(?P<months>\d{2})(?P<days>\d{2}) (?P<hours>\d{2})(?P<minutes>\d{2})(?P<seconds>\d{2})$`)
)

func (r WhenRotate) lower() WhenRotate { return WhenRotate(strings.ToLower(string(r))) }

// interval returns the duration of an interval in whenRotate, given the time
func (r WhenRotate) interval(t time.Time) time.Duration {
	switch r {
	case Hour:
		return 1 * time.Hour
	case Day:
		return oneDay
	case Month:
		return time.Duration(daysIn(t.Month(), t.Year())) * oneDay
	case Year:
		return oneYear
	default:
		// Default, should not reach here
		return oneDay
	}
}

// daysIn returns the number of days in a month for a given year.
func daysIn(m time.Month, year int) int {
	return time.Date(year, m+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// valid returns an error if its not valid
func (r WhenRotate) valid() error {
	switch r {
	case Hour, Day, Month, Year:
		return nil
	default:
		return fmt.Errorf("invalid rotation interval specified: %s", r)
	}
}

// baseRotateTime returns a sensible default time offset for rotating.
func (r WhenRotate) baseRotateTime() timeSchedule {
	var off timeSchedule
	switch r {
	case Hour, Day:
		return off
	case Month:
		off.day = 1
		return off
	case Year:
		off.day = 1
		off.month = 1
		return off
	default:
		off.day = 1
		off.month = 1
		return off
	}
}

// parseTimeSchedule parses the time offset passed in such that they at least make
// some sense relative to the current When.
// For example if When = "d", then an offset of 250000 does not make sense as
// a day only has a maximum of 24 hours
// This does not handle year offset specifically for the month,
// it just takes an upper bound of the max number of days a month has (i.e. 31 days),
// so for When = "y", "0231 150405" will still be considered valid.
func (r WhenRotate) parseTimeSchedule(offsetStr string) (timeSchedule, error) { //nolint:gocyclo // Let cyclo err here go
	var offsetRegex *regexp.Regexp
	when := r
	switch when {
	case Hour:
		offsetRegex = hourOffsetRegex
	case Day:
		offsetRegex = dayOffsetRegex
	case Month:
		offsetRegex = monthOffsetRegex
	case Year:
		offsetRegex = yearOffsetRegex
	default:
		return timeSchedule{}, fmt.Errorf("invalid rotation interval specified: %s, expected %v", r, [...]WhenRotate{Hour, Day, Month, Year})
	}
	match := offsetRegex.FindStringSubmatch(offsetStr)
	if len(match) != len(offsetRegex.SubexpNames()) {
		validFormatMsg := map[WhenRotate]string{
			Hour:  `"0405" (MMSS)`,
			Day:   `"150405" (HHMMSS)`,
			Month: `"02 150405" (DD HHMMSS)`,
			Year:  `"0102 150405" (mmDD HHMMSS)`,
		}
		return timeSchedule{}, fmt.Errorf("invalid offset passed in for 'when' value '%s', expected value of format %s, got '%s'", r, validFormatMsg[when], offsetStr)
	}
	var off timeSchedule
	for i, name := range offsetRegex.SubexpNames() {
		if i == 0 {
			continue
		}
		// Ignore the error here, the regex should have handled it properly here
		res, _ := strconv.Atoi(match[i])
		switch name {
		case "months":
			if res < 1 || res > 12 {
				return timeSchedule{}, fmt.Errorf("invalid month offset %d, month must be between 1-12", res)
			}
			off.month = res
		case "days":
			if res < 1 || res > 31 {
				return timeSchedule{}, fmt.Errorf("invalid day offset %d, day must be between 1-31", res)
			}
			off.day = res
		case "hours":
			if res < 0 || res > 23 {
				return timeSchedule{}, fmt.Errorf("invalid hour offset %d, hour must be between 0-23", res)
			}
			off.hour = res
		case "minutes":
			if res < 0 || res > 59 {
				return timeSchedule{}, fmt.Errorf("invalid minute offset %d, minute must be between 0-59", res)
			}
			off.minute = res
		case "seconds":
			if res < 0 || res > 59 {
				return timeSchedule{}, fmt.Errorf("invalid second offset %d, second must be between 0-59", res)
			}
			off.second = res
		}
	}
	return off, nil
}

// offsetCurrentTime returns the offset time from the current time passed in.
func (r WhenRotate) offsetCurrentTime(currentTime time.Time, sch timeSchedule) time.Time {
	year, month, day := currentTime.Date()
	hour := currentTime.Hour()
	loc := currentTime.Location()
	switch r {
	case Hour:
		return time.Date(year, month, day, hour, sch.minute, sch.second, 0, loc)
	case Day:
		return time.Date(year, month, day, sch.hour, sch.minute, sch.second, 0, loc)
	case Month:
		return time.Date(year, month, sch.day, sch.hour, sch.minute, sch.second, 0, loc)
	case Year:
		return time.Date(year, time.Month(sch.month), sch.day, sch.hour, sch.minute, sch.second, 0, loc)
	default:
		return currentTime
	}
}

// AddTime adds n Hours/Days/Months/Years depending on WhenRotate
func (r WhenRotate) AddTime(t time.Time, n int) time.Time {
	switch r {
	case Hour:
		return t.Add(time.Duration(n) * time.Hour)
	case Day:
		return t.AddDate(0, 0, n)
	case Month:
		return t.AddDate(0, n, 0)
	case Year:
		return t.AddDate(n, 0, 0)
	default:
		return t
	}
}

// calcRotationTimes calculates the next and previous rotation times based on
// the timeRotationSchedule.
// This function ignores any potential problems with daylight savings
func (r WhenRotate) calcRotationTimes(t time.Time, timeSchedules []timeSchedule) (prev, next time.Time) {
	// Check first offset time first by picking out the last entry and minus 1 Hour/Day/Month/Year
	firstOffsetToCheck := r.AddTime(r.offsetCurrentTime(t, timeSchedules[len(timeSchedules)-1]), -1)
	if firstOffsetToCheck.After(t) {
		return prev, firstOffsetToCheck
	}
	var lastOffsetToCheck time.Time
	next = firstOffsetToCheck
	for i, sch := range timeSchedules {
		prev = next
		next = r.offsetCurrentTime(t, sch)
		if i == 0 {
			// last offset entry to check is the 1st offset time but add 1 Hour/Day/Month/Year
			lastOffsetToCheck = r.AddTime(next, 1)
		}
		if !next.After(t) {
			continue
		}
		return prev, next
	}
	if lastOffsetToCheck.After(t) {
		return next, lastOffsetToCheck
	}
	// Code should not reach here, if it did anyway it will move the date
	// forward by 1 * (when), and prev will be assumed to be - 1 * (when)
	return t.Add(-r.interval(t)), t.Add(r.interval(t))
}

// timeSchedule is the rough schedule of when to rotate. By itself this struct
// has no meaning, it needs to be paired with WhenRotate.
type timeSchedule struct {
	year   int
	month  int
	day    int
	hour   int
	minute int
	second int
}

func (t *timeSchedule) approxDuration() time.Duration {
	return time.Duration(t.year)*oneYear +
		time.Duration(t.month)*approxOneMonth +
		time.Duration(t.day)*oneDay +
		time.Duration(t.hour)*time.Hour +
		time.Duration(t.minute)*time.Minute +
		time.Duration(t.second)*time.Second
}

// timeSchedules is a slice of timeSchedules, it satisfies sort.Interface
type timeSchedules []timeSchedule

// Len is the number of elements in timeSchedules.
func (s timeSchedules) Len() int { return len(s) }

// Less tells is timeSchedules[i] comes before timeSchedules[j]. We sort in an ascending order.
func (s timeSchedules) Less(i, j int) bool {
	return s[i].approxDuration() < s[j].approxDuration()
}

// Swap swaps the elements with indexes i and j.
func (s timeSchedules) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
