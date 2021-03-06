// package logfeller implements a library for writing to and rotating files
// based on a timed schedule.
package logfeller

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// File is the rotational file handler. It writes to the filename specified
// and will rotate based on the schedule passed in and the when field.
type File struct {
	// Filename is the filename to write to. Uses the filename
	// `<cmdname>-logfeller.log` in os.TempDir() if empty.
	Filename string `json:"filename" yaml:"filename"`
	// When tells the logger to rotate the file, it is case insensitive.
	// Currently supported values are
	// 	"h" - hour
	// 	"d" - day
	// 	"m" - month
	// 	"y" - year
	When WhenRotate `json:"when" yaml:"when"`
	// RotationSchedule defines the exact time that the rotator should be
	// rotating. The values that should be passed into depends on the When field.
	// If When is:
	// 	"h" - pass in strings of format "0405" (MMSS)
	// 	"d" - pass in strings of format "150405" (HHMMSS)
	// 	"m" - pass in strings of format "02 150405" (DD HHMMSS)
	// 	"y" - pass in strings of format "0102 150405" (mmDD HHMMSS)
	// where mm, DD, HH, MM, SS represents month, day, hour, minute
	// and seconds respectively.
	// If RotationSchedule is empty, a sensible default will be used instead.
	RotationSchedule []string `json:"rotation_schedule" yaml:"rotation-schedule"`
	// UseLocal determines if the time used to rotate is based on the system's
	// local time
	UseLocal bool `json:"use_local" yaml:"use-local"`
	// Backups maintains the number of backups to keep. If this is empty, do
	// not delete backups.
	Backups int `json:"backups" yaml:"backups"`
	// BackupTimeFormat is the backup time format used when logfeller rotates
	// the file. Defaults to "2006-01-02.150405" if empty
	// See the golang `time` package for more example formats
	// https://golang.org/pkg/time/#Time.Format
	BackupTimeFormat string `json:"backup_time_format" yaml:"backup-time-format"`

	// timeRotationSchedule stores the parsed rotational schedule.
	// These times are sorted and behave more like offsets instead.
	// see File.nextAndPrevRotateTime()
	timeRotationSchedule []timeOffset
	// possibleNextTimes is a buffer that is reused everytime nextAndPrevRotateTime
	// is called. This is to minimise allocations as we know beforehand the
	// number of possibleNextTimes = len(timeRotationSchedule) + 2.
	possibleNextTimes []time.Time

	// TODO: Implement the actual io.WriteCloser using the below fields
	// // mu protects the following fields below
	// mu           sync.Mutex
	// rotateAt     time.Time
	// prevRotateAt time.Time
	// file         *os.File

	initOnce sync.Once
}

const defaultBackupTimeFormat = "2006-01-02.150405"

func (f *File) init() error {
	var err error
	f.initOnce.Do(func() {
		if f.Filename == "" {
			name := filepath.Base(os.Args[0]) + "-logfeller.log"
			f.Filename = filepath.Join(os.TempDir(), name)
		}
		f.When = WhenRotate(strings.ToLower(string(f.When)))
		if errInner := f.When.valid(); errInner != nil {
			err = fmt.Errorf("logfeller: init failed, %w", errInner)
			return
		}
		for _, schedule := range f.RotationSchedule {
			off, errInner := f.When.parseTimeoffset(schedule)
			if errInner != nil {
				err = fmt.Errorf("logfeller: failed to parse rotation schedule \"%s\": %w", schedule, errInner)
				return
			}
			f.timeRotationSchedule = append(f.timeRotationSchedule, off)
		}
		if len(f.timeRotationSchedule) == 0 {
			// If f.timeRotationSchedule is empty, add in a sensible default for
			// rotation.
			f.timeRotationSchedule = append(f.timeRotationSchedule, f.When.baseRotateTime())
		}
		// Sort in ascending order for rotation
		sort.Sort(timeOffsets(f.timeRotationSchedule))
		// add in before and after as list of possibleNextTimes
		extraNextTimes := 2
		f.possibleNextTimes = make([]time.Time, len(f.timeRotationSchedule)+extraNextTimes)
		if f.BackupTimeFormat == "" {
			f.BackupTimeFormat = defaultBackupTimeFormat
		}
	})
	return err
}

// UnmarshalJSON unmarshals JSON to the file handler and init f too.
func (f *File) UnmarshalJSON(data []byte) error {
	err := json.Unmarshal(data, f)
	if err != nil {
		return err
	}
	return f.init()
}

// TODO: IMPLEMENT YAML UNMARSHALLING
// func (f *File) UnmarshalYAML(unmarshal func(interface{}) error) error {
// return nil
// }

// WhenRotate helps reason about logic related to rotation of the file.
type WhenRotate string

// valid returns an error if its not valid
func (r WhenRotate) valid() error {
	switch WhenRotate(strings.ToLower(string(r))) {
	case Hour, Day, Month, Year:
		return nil
	default:
		return fmt.Errorf("invalid rotation interval specified: %s", r)
	}
}

// baseRotateTime returns a sensible default time offset for rotating.
func (r WhenRotate) baseRotateTime() timeOffset {
	var off timeOffset
	switch WhenRotate(strings.ToLower(string(r))) {
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

// parseTimeoffset parses the time offset passed in such that they at least make
// some sense relative to the current When.
// For example if When = "d", then an offset of 250000 does not make sense as
// a day only has a maximum of 24 hours
// This does not handle year offset specifically for the month,
// it just takes an upper bound of the max number of days a month has (i.e. 31 days),
// so for When = "y", "0231 150405" will still be considered valid.
func (r WhenRotate) parseTimeoffset(offsetStr string) (timeOffset, error) { //nolint:gocyclo // Let cyclo err here go
	var offsetRegex *regexp.Regexp
	when := WhenRotate(strings.ToLower(string(r)))
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
		return timeOffset{}, fmt.Errorf("invalid rotation interval specified: %s, expected %v", r, [...]WhenRotate{Hour, Day, Month, Year})
	}
	match := offsetRegex.FindStringSubmatch(offsetStr)
	if len(match) != len(offsetRegex.SubexpNames()) {
		validFormatMsg := map[WhenRotate]string{
			Hour:  `"0405" (MMSS)`,
			Day:   `"150405" (HHMMSS)`,
			Month: `"02 150405" (DD HHMMSS)`,
			Year:  `"0102 150405" (mmDD HHMMSS)`,
		}
		return timeOffset{}, fmt.Errorf("invalid offset passed in for 'when' value '%s', expected value of format %s, got '%s'", r, validFormatMsg[when], offsetStr)
	}
	var off timeOffset
	for i, name := range offsetRegex.SubexpNames() {
		if i == 0 {
			continue
		}
		// Ignore the error here, the regex should have handled it properly here
		res, _ := strconv.Atoi(match[i])
		switch name {
		case "months":
			if res < 1 || res > 12 {
				return timeOffset{}, fmt.Errorf("invalid month offset %d, month must be between 1-12", off.month)
			}
			off.month = res
		case "days":
			if res < 1 || res > 31 {
				return timeOffset{}, fmt.Errorf("invalid day offset %d, day must be between 1-31", off.day)
			}
			off.day = res
		case "hours":
			if res < 0 || res > 23 {
				return timeOffset{}, fmt.Errorf("invalid hour offset %d, hour must be between 0-23", off.hour)
			}
			off.hour = res
		case "minutes":
			if res < 0 || res > 59 {
				return timeOffset{}, fmt.Errorf("invalid minute offset %d, minute must be between 0-59", off.minute)
			}
			off.minute = res
		case "seconds":
			if res < 0 || res > 59 {
				return timeOffset{}, fmt.Errorf("invalid second offset %d, second must be between 0-59", off.second)
			}
			off.second = res
		}
	}
	return off, nil
}

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

type timeOffset struct {
	year   int
	month  int
	day    int
	hour   int
	minute int
	second int
	sec    int
}

// timeOffsets is a slice of timeOffsets, it satisfies sort.Interface
type timeOffsets []timeOffset

// Len is the number of elements in timeOffsets.
func (offs timeOffsets) Len() int { return len(offs) }

// Less tells is timeOffsets[i] comes before timeOffsets[j]. We sort in an ascending order.
func (offs timeOffsets) Less(i, j int) bool {
	switch {
	case offs[i].year < offs[j].year:
		return true
	case offs[i].month < offs[j].month:
		return true
	case offs[i].day < offs[j].day:
		return true
	case offs[i].hour < offs[j].hour:
		return true
	case offs[i].minute < offs[j].minute:
		return true
	case offs[i].second < offs[j].second:
		return true
	case offs[i].sec < offs[j].sec:
		return true
	}
	return false
}

// Swap swaps the elements with indexes i and j.
func (offs timeOffsets) Swap(i, j int) { offs[i], offs[j] = offs[j], offs[i] }
