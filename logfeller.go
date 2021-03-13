/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

// package logfeller implements a library for writing to and rotating files
// based on a timed schedule. This project is inspired by
// https://github.com/natefinch/lumberjack but serves a different niche.
// Logfeller handles which file to write to based not on
// max size but on a schedule (such as every day at 12am etc.)
//
// As with lumberjack, logfeller is intended to be a pluggable component in a
// logging stack that controls how files are written and rotated.
//
// Logfeller works with any package that can write to an io.Writer, such
// as the standard library's log package.
package logfeller

import (
	"encoding/json"
	"fmt"
	"io"
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
	nowFunc  func() time.Time
}

const (
	defaultBackupTimeFormat               = ".2006-01-02T1504-05"
	fileOpenMode              os.FileMode = 0644
	dirCreateMode             os.FileMode = 0755
	fileWriteCreateAppendFlag             = os.O_WRONLY | os.O_CREATE | os.O_APPEND
	fileWriteAppend                       = os.O_WRONLY | os.O_APPEND
	oneMB                                 = 1024 * 1024
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
		if f.When == "" {
			f.When = Day
		} else {
			f.When = f.When.lower()
		}
		if errInner := f.When.valid(); errInner != nil {
			f.initErr = fmt.Errorf("logfeller: init failed, %v", errInner)
			return
		}
		// Populate the rotation schedule offsets
		f.timeRotationSchedule = make([]timeSchedule, 0, len(f.RotationSchedule))
		for _, schedule := range f.RotationSchedule {
			sch, errInner := f.When.parseTimeSchedule(schedule)
			if errInner != nil {
				f.initErr = fmt.Errorf("logfeller: failed to parse rotation schedule \"%s\": %v", schedule, errInner)
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
		if f.nowFunc == nil {
			f.setNowFunc(time.Now)
		}
	})
	return f.initErr
}

// setNowFunc sets the nowFunc f uses to determine filenames, rotation times
// etc. This function is used to mock out the time function used such that
// we can have control over it in tests.
func (f *File) setNowFunc(nf func() time.Time) { f.nowFunc = nf }

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

func (f *File) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type alias File
	// Replace f with tmp and unmarshal there to prevent infinite loops
	tmp := (*alias)(f)
	err := unmarshal(tmp)
	if err != nil {
		return err
	}
	return f.init()
}

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
		return fmt.Errorf("rotate close error: %v", err)
	}
	if err := f.rotateOpen(); err != nil {
		return fmt.Errorf("rotate open error: %v", err)
	}
	if err := f.triggerTrim(); err != nil {
		return err
	}
	return nil
}

// Rotate closes the existing log file and flushes its content to backup.
// new one. This is a helper function for applications to flush logs to backup.
func (f *File) Rotate() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.rotate()
}

func (f *File) openExistingOrNew() error {
	if err := f.triggerTrim(); err != nil {
		return err
	}
	fileInfo, err := os.Stat(f.Filename)
	if os.IsNotExist(err) {
		// If opening something new that previously didnt exist, we rotate
		// based on current time.
		f.updateRotateAt(f.calcRotationTimes(f.nowFunc()))
		return f.rotateOpen()
	}
	if err != nil {
		return fmt.Errorf("error getting file info: %v", err)
	}
	// file exists, update rotate at based on file's modified time and check if should rotate
	f.updateRotateAt(f.calcRotationTimes(fileInfo.ModTime()))
	err = f.checkAndRotate()
	if err == nil && f.file != nil {
		return nil
	}
	// did not rotate, set try to set file
	fh, err := os.OpenFile(f.Filename, fileWriteCreateAppendFlag, fileOpenMode)
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
	return f.time(f.nowFunc()).After(f.rotateAt)
}

func (f *File) checkAndRotate() error {
	if f.shouldRotate() {
		err := f.rotate()
		f.updateRotateAt(f.calcRotationTimes(f.nowFunc()))
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
		if originalFileExistAndIsNotEmpty {
			// original file exists and its not empty, ready to be rotated
			if os.IsNotExist(err2) {
				// If dst doesnt exist, move orignal file to dst path.
				if err := os.Rename(f.Filename, dstFilename); err != nil {
					return fmt.Errorf("unable to rename file %s to %s with err: %v", f.Filename, dstFilename, err)
				}
			}
			if err2 == nil {
				// If dstfilename is found somehow, we flush current file's content
				// to this dst file
				dstFile, err := os.OpenFile(dstFilename, fileWriteAppend, mode)
				if err != nil {
					return fmt.Errorf("open existing dst file %s to append fail with err: %v", dstFilename, err)
				}
				file, err := os.Open(f.Filename)
				if err != nil {
					return fmt.Errorf("open file %s to append to existing dst fail with err: %v", f.Filename, err)
				}
				buf := make([]byte, oneMB)
				_, err = io.CopyBuffer(dstFile, file, buf)
				if err != nil {
					return fmt.Errorf("copy append from file %s to dst %s fail with error: %v", f.Filename, dstFilename, err)
				}
				dstFile.Close()
				file.Close()
				// Remove the existing file after appending, we ignore the error here
				_ = os.Remove(f.Filename)
			}
		}
	}
	fh, err := os.OpenFile(f.Filename, fileWriteCreateAppendFlag, mode)
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
	firstOffsetToCheck := r.addTime(r.nearestScheduledTime(t, timeSchedules[len(timeSchedules)-1]), -1)
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
			lastOffsetToCheck = r.addTime(next, 1)
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
// then the resultant filename will be /var/www/some-app/info<timstamp>.log
// It uses the timstamp format from f.BackupTimeFormat.
func (f *File) filenameWithTimestamp(t time.Time) string {
	timestamp := t.Format(f.BackupTimeFormat)
	return filepath.Join(f.directory, fmt.Sprint(f.fileBase, timestamp, f.ext))
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
	dirEntries, err := ioutil.ReadDir(f.directory)
	if err != nil {
		return fmt.Errorf("cannot read log file directory %s: %v", f.directory, err)
	}
	type fileInfoWithTime struct {
		t time.Time
		os.FileInfo
	}
	var backupFIs []fileInfoWithTime
	for _, dirEntry := range dirEntries {
		if dirEntry.IsDir() {
			continue
		}
		filename := dirEntry.Name()
		if !strings.HasPrefix(filename, f.fileBase) || !strings.HasSuffix(filename, f.ext) {
			// file is not a backup file if the fileBase and ext dont match
			continue
		}
		// get time from filename
		timestamp := strings.TrimSuffix(strings.TrimPrefix(filename, f.fileBase), f.ext)
		t, err := time.Parse(f.BackupTimeFormat, timestamp)
		if err != nil {
			continue
		}
		backupFIs = append(backupFIs, fileInfoWithTime{t, dirEntry})
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
