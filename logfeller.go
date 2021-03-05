// package logfeller implements a library for writing to and rotating files
// based on a timed schedule.
package logfeller

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
	BackupTimeFormat string

	// timeRotationSchedule stores the rotational schedule that is parsed into
	// time.Time. These times are sorted and behave more like offsets instead.
	// see File.nextAndPrevRotateTime()
	timeRotationSchedule []time.Time
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
		dtf := f.When.DateTimeFormat()
		for _, schedule := range f.RotationSchedule {
			t, errInner := time.Parse(dtf, schedule)
			if errInner != nil {
				err = fmt.Errorf("logfeller: failed to parse rotation schedule \"%s\": %w", schedule, errInner)
				return
			}
			f.timeRotationSchedule = append(f.timeRotationSchedule, t)
		}
		if len(f.timeRotationSchedule) == 0 {
			// If f.timeRotationSchedule is empty, add in a sensible default for
			// rotation.
			f.timeRotationSchedule = append(f.timeRotationSchedule, f.When.baseRotateTime())
		}
		// Sort in ascending order for rotation
		sort.Slice(f.timeRotationSchedule, func(i, j int) bool {
			return f.timeRotationSchedule[i].Before(f.timeRotationSchedule[j])
		})
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
	switch r {
	case Hour, Day, Month, Year:
		return nil
	default:
		return fmt.Errorf("invalid rotation interval specified: %s", r)
	}
}

// baseRotateTime returns a sensible default time for rotating.
func (r WhenRotate) baseRotateTime() time.Time {
	dateFormat := r.DateTimeFormat()
	var defaultTimeString string
	switch r {
	case Hour:
		defaultTimeString = "0000"
	case Day:
		defaultTimeString = "000000"
	case Month:
		defaultTimeString = "01 000000"
	case Year:
		defaultTimeString = "0101 000000"
	default:
		// Default, should not reach here
		defaultTimeString = "00000101 000000"
	}
	t, _ := time.Parse(dateFormat, defaultTimeString)
	return t
}

// DateTimeFormat determines the desired format of date to parse the rotation
// schedule in.
func (r WhenRotate) DateTimeFormat() string {
	switch r {
	case Hour:
		return "0405"
	case Day:
		return "150405"
	case Month:
		return "02 150405"
	case Year:
		return "0102 150405"
	default:
		// Default, should not reach here
		return "20060102 150405"
	}
}

const (
	Hour  WhenRotate = "h"
	Day   WhenRotate = "d"
	Month WhenRotate = "m"
	Year  WhenRotate = "y"
)
