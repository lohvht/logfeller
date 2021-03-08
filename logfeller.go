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
	// 	"h" - pass in strings of format "04:05" (MM:SS)
	// 	"d" - pass in strings of format "1504:05" (HHMM:SS)
	// 	"m" - pass in strings of format "02 1504:05" (DD HHMM:SS)
	// 	"y" - pass in strings of format "0102 1504:05" (mmDD HHMM:SS)
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
	// the file. Defaults to ".2006-01-02T1504-05" if empty
	// See the golang `time` package for more example formats
	// https://golang.org/pkg/time/#Time.Format
	BackupTimeFormat string `json:"backup_time_format" yaml:"backup-time-format"`

	// timeRotationSchedule stores the parsed rotational schedule.
	// These offsets are sorted.
	// This field is populated on init()
	timeRotationSchedule []timeSchedule
	// directory is the directory of the current Filename
	// This field is populated on init()
	directory string
	// fileBase is the base name of the file without extension
	// This field is populated on init()
	fileBase string
	// ext is the file's extension.
	// This field is populated on init()
	ext string

	rotateAt     time.Time
	prevRotateAt time.Time

	initOnce sync.Once
}

const (
	defaultBackupTimeFormat             = ".2006-01-02T1504-05"
	fileOpenMode            os.FileMode = 0644
	dirCreateMode           os.FileMode = 0755
	fileFlag                            = os.O_WRONLY | os.O_CREATE | os.O_APPEND
)

func (f *File) init() error {
	var err error
	f.initOnce.Do(func() {
		if f.Filename == "" {
			basename := filepath.Base(os.Args[0])
			trimmedCmdName := strings.TrimSuffix(basename, filepath.Ext(basename))
			name := trimmedCmdName + "-logfeller.log"
			f.Filename = filepath.Join(os.TempDir(), name)
		}
		baseFilename := filepath.Base(f.Filename)
		f.directory = filepath.Dir(f.Filename)
		f.ext = filepath.Ext(baseFilename)
		// get the base file name without extensions
		f.fileBase = baseFilename[:len(baseFilename)-len(f.ext)]
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

// Time handles time for File.
func (f *File) Time(t time.Time) time.Time {
	if !f.UseLocal {
		return t.UTC()
	}
	return t
}

func (f *File) updateRotateAt(prevRotateAt, rotateAt time.Time) {
	f.prevRotateAt = prevRotateAt
	f.rotateAt = rotateAt
}
