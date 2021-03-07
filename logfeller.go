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
	// These offsets are sorted.
	// This field is populated on init()
	timeRotationSchedule []timeSchedule

	// directory is the directory of the current Filename
	// This field is populated in init()
	directory string

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
			basename := filepath.Base(os.Args[0])
			trimmedCmdName := strings.TrimSuffix(basename, filepath.Ext(basename))
			name := trimmedCmdName + "-logfeller.log"
			f.Filename = filepath.Join(os.TempDir(), name)
		}
		f.directory = filepath.Dir(f.Filename)
		f.When = f.When.lower()
		if errInner := f.When.valid(); errInner != nil {
			err = fmt.Errorf("logfeller: init failed, %w", errInner)
			return
		}
		// Populate the rotation schedule offsets
		f.timeRotationSchedule = make([]timeSchedule, 0, len(f.RotationSchedule))
		for _, schedule := range f.RotationSchedule {
			sch, errInner := f.When.parseTimeSchedule(schedule)
				if errInner != nil {
					err = fmt.Errorf("logfeller: failed to parse rotation schedule \"%s\": %w", schedule, errInner)
					return
				}
			f.timeRotationSchedule = append(f.timeRotationSchedule, sch)
			}
		if len(f.RotationSchedule) == 0 {
			f.timeRotationSchedule = append(f.timeRotationSchedule, f.When.baseRotateTime())
		}
		sort.Sort(timeSchedules(f.timeRotationSchedule))
		if f.BackupTimeFormat == "" {
			f.BackupTimeFormat = defaultBackupTimeFormat
		}
	})
	return err
}

// UnmarshalJSON unmarshals JSON to the file handler and init f too.
func (f *File) UnmarshalJSON(data []byte) error {
	type alias File
	// Replace f with tmp and unmarshal there to prevent infinite loops
	tmp := (*alias)(f)
	err := json.Unmarshal(data, tmp)
	if err != nil {
		return err
	}
	return f.init()
}

// TODO: IMPLEMENT YAML UNMARSHALLING
// func (f *File) UnmarshalYAML(unmarshal func(interface{}) error) error {
// return nil
// }
