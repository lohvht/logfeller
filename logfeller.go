/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

// package logfeller implements a library for writing to and rotating files
// based on a timed schedule.
package logfeller

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
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
	ext    string
	trimCh chan struct{}

	// mu protects the following fields below
	mu           sync.Mutex
	rotateAt     time.Time
	prevRotateAt time.Time
	file         *os.File

	initOnce sync.Once
	initErr  error
}

const (
	defaultBackupTimeFormat             = ".2006-01-02T1504-05"
	fileOpenMode            os.FileMode = 0644
	dirCreateMode           os.FileMode = 0755
	fileFlag                            = os.O_WRONLY | os.O_CREATE | os.O_APPEND
)

func (f *File) init() error {
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
			f.initErr = fmt.Errorf("logfeller: init failed, %w", errInner)
			return
		}
		// Populate the rotation schedule offsets
		f.timeRotationSchedule = make([]timeSchedule, 0, len(f.RotationSchedule))
		for _, schedule := range f.RotationSchedule {
			sch, errInner := f.When.parseTimeSchedule(schedule)
			if errInner != nil {
				f.initErr = fmt.Errorf("logfeller: failed to parse rotation schedule \"%s\": %w", schedule, errInner)
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
		f.trimCh = make(chan struct{}, 1)
		go func() {
			for range f.trimCh {
				_ = f.trim()
			}
		}()
	})
	return f.initErr
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

// Write implements io.Writer. It first checks if it should rotate first before
// writing.
func (f *File) Write(p []byte) (int, error) {
	if err := f.init(); err != nil {
		return 0, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.file == nil {
		if err := f.openExistingOrNew(); err != nil {
			return 0, err
		}
	}
	if err := f.checkAndRotate(); err != nil {
		return 0, err
	}
	return f.file.Write(p)
}

// Sync commits file content.
func (f *File) Sync() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.file == nil {
		return nil
	}
	return f.file.Sync()
}

// Close implements io.Closer, and closes the current file.
func (f *File) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.close()
}

// close closes the file if it is open.
// sets file to nil.
func (f *File) close() error {
	if f.file == nil {
		return nil
	}
	err := f.file.Close()
	f.file = nil
	return err
}

// rotate closes the file and rotates it after that.
func (f *File) rotate() error {
	if err := f.close(); err != nil {
		return fmt.Errorf("rotate close error: %w", err)
	}
	if err := f.rotateOpen(); err != nil {
		return fmt.Errorf("rotate open error: %w", err)
	}
	return nil
}

func (f *File) openExistingOrNew() error {
	if err := f.triggerTrim(); err != nil {
		return err
	}
	fileInfo, err := os.Stat(f.Filename)
	if os.IsNotExist(err) {
		// If opening something new that previously didnt exist, we rotate
		// based on current time.
		f.updateRotateAt(f.calcRotationTimes(time.Now()))
		return f.rotateOpen()
	}
	if err != nil {
		return fmt.Errorf("error getting file info: %w", err)
	}
	// file exists, update rotate at based on file's modified time and check if should rotate
	f.updateRotateAt(f.calcRotationTimes(fileInfo.ModTime()))
	err = f.checkAndRotate()
	if err != nil {
		return err
	}
	// did not rotate, set try to set file
	fh, err := os.OpenFile(f.Filename, fileFlag, fileOpenMode)
	if err != nil {
		// last resort
		return f.rotateOpen()
	}
	f.file = fh
	return nil
}

// time handles time for File.
func (f *File) time(t time.Time) time.Time {
	if !f.UseLocal {
		return t.UTC()
	}
	return t
}

func (f *File) shouldRotate() bool {
	return f.time(time.Now()).After(f.rotateAt)
}

func (f *File) checkAndRotate() error {
	if f.shouldRotate() {
		err := f.rotate()
		f.updateRotateAt(f.calcRotationTimes(time.Now()))
		return err
	}
	return nil
}

// rotateOpen moves any existing log file and opens a new log file for writing.
// This function assumes that the original file has already been closed.
func (f *File) rotateOpen() error {
	if err := os.MkdirAll(f.directory, dirCreateMode); err != nil {
		return fmt.Errorf("cannot make directories for new logfiles at %s: %v", f.Filename, err)
	}
	mode := fileOpenMode
	if info, err := os.Stat(f.Filename); err == nil && info.Size() > 0 {
		// TODO: Potentially need a file locking mechanism here otherwise
		// writes and deletes may not be correctly synchronised.
		mode = info.Mode()
		// use prevRotateAt as the log was for the previous day
		dstFilename := f.filenameWithTimestamp(f.time(f.prevRotateAt))
		originalFilestat, err1 := os.Stat(f.Filename)
		_, err2 := os.Stat(dstFilename)
		originalFileExistAndIsNotEmpty := err1 == nil && originalFilestat.Size() > 0
		if originalFileExistAndIsNotEmpty && os.IsNotExist(err2) {
			if err := os.Rename(f.Filename, dstFilename); err != nil {
				return fmt.Errorf("unable to rename logfile %s to %s with err: %w", f.Filename, dstFilename, err)
			}
		}
	}
	fh, err := os.OpenFile(f.Filename, fileFlag, mode)
	if err != nil {
		return err
	}
	f.file = fh
	return nil
}

// calcRotationTimes calculates the next and previous rotation times based on
// the timeRotationSchedule.
// This function ignores any potential problems with daylight savings
func (f *File) calcRotationTimes(t time.Time) (prev, next time.Time) {
	t = f.time(t)
	r := f.When
	timeSchedules := f.timeRotationSchedule
	// Check first offset time first by picking out the last entry and minus 1 Hour/Day/Month/Year
	firstOffsetToCheck := r.AddTime(r.nearestScheduledTime(t, timeSchedules[len(timeSchedules)-1]), -1)
	if firstOffsetToCheck.After(t) {
		return prev, firstOffsetToCheck
	}
	var lastOffsetToCheck time.Time
	next = firstOffsetToCheck
	for i, sch := range timeSchedules {
		prev = next
		next = r.nearestScheduledTime(t, sch)
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

// filenameWithTimestamp returns a new filename with timestamps from the given
// time t passed in. If the filename was /var/www/some-app/info.log,
// then the resultant filename will be /var/www/some-app/info-<timstamp>.log
// It uses the timstamp format from f.BackupTimeFormat.
func (f *File) filenameWithTimestamp(t time.Time) string {
	timestamp := t.Format(f.BackupTimeFormat)
	return filepath.Join(f.directory, fmt.Sprint(f.fileBase, "-", timestamp, f.ext))
}

// updateRotateAt updates prevRotateAt and rotateAt
func (f *File) updateRotateAt(prevRotateAt, rotateAt time.Time) {
	f.prevRotateAt = prevRotateAt
	f.rotateAt = rotateAt
}

// triggerTrim the trimming process via trimCh
func (f *File) triggerTrim() error {
	if err := f.init(); err != nil {
		return err
	}
	f.trimCh <- struct{}{}
	return nil
}

// trim does the cleanup of rotated backup files
func (f *File) trim() error {
	if f.Backups <= 0 {
		return nil
	}
	fileInfos, err := ioutil.ReadDir(f.directory)
	if err != nil {
		return fmt.Errorf("cannot read log file directory %s: %w", f.directory, err)
	}
	type fileInfoWithTime struct {
		t time.Time
		fs.FileInfo
	}
	var backupFIs []fileInfoWithTime
	for _, fi := range fileInfos {
		if fi.IsDir() {
			continue
		}
		filename := fi.Name()
		if !strings.HasPrefix(filename, f.fileBase) || strings.HasSuffix(filename, f.ext) {
			// file is not a backup file if the fileBase and ext dont match
			continue
		}
		// get time from filename
		timestamp := strings.TrimSuffix(strings.TrimPrefix(filename, f.fileBase), f.ext)
		t, err := time.Parse(f.BackupTimeFormat, timestamp)
		if err != nil {
			continue
		}
		backupFIs = append(backupFIs, fileInfoWithTime{t, fi})
	}
	sort.SliceStable(backupFIs, func(i, j int) bool { return backupFIs[i].t.After(backupFIs[j].t) })

	var toRemove []fileInfoWithTime
	if len(backupFIs) > f.Backups {
		toRemove = backupFIs[f.Backups:]
	}
	var errs multipleErrors
	for _, fi := range toRemove {
		err := os.Remove(filepath.Join(f.directory, fi.Name()))
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

type multipleErrors []error

func (errs multipleErrors) Error() string {
	if len(errs) == 1 {
		return errs[0].Error()
	}
	var sb strings.Builder
	sb.WriteString("errors :")
	for i, err := range errs {
		sb.WriteString(err.Error())
		if i < len(errs)-1 {
			sb.WriteString(";")
		}
	}
	return sb.String()
}
