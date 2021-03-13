/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

// package logfeller implements a library for writing to and rotating files
// based on a timed schedule.
package logfeller

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/lohvht/logfeller/internal/testutils"
)

func TestFile_init(t *testing.T) {
	type wantFields struct {
		Filename             string
		When                 WhenRotate
		RotationSchedule     []string
		UseLocal             bool
		Backups              int
		BackupTimeFormat     string
		timeRotationSchedule []timeSchedule
		directory            string
		fileBase             string
		ext                  string
	}
	tests := []struct {
		name    string
		f       *File
		want    wantFields
		wantErr bool
	}{
		{
			name: "all_fields_specified",
			f: &File{
				Filename:         "file.txt",
				When:             "H",
				RotationSchedule: []string{"14:30", "12:00"},
				UseLocal:         true,
				Backups:          40,
				BackupTimeFormat: "Jan _2 15:04:05",
			},
			want: wantFields{
				Filename:         "file.txt",
				When:             "h",
				RotationSchedule: []string{"14:30", "12:00"},
				UseLocal:         true,
				Backups:          40,
				BackupTimeFormat: "Jan _2 15:04:05",
				timeRotationSchedule: []timeSchedule{
					{minute: 12},
					{minute: 14, second: 30},
				},
				directory: ".",
				fileBase:  "file",
				ext:       ".txt",
			},
		},
		{
			name: "omit_all_fields",
			f:    &File{},
			want: wantFields{
				Filename:         filepath.Join(os.TempDir(), "logfeller.test-logfeller.log"),
				When:             "d",
				BackupTimeFormat: ".2006-01-02T1504-05",
				timeRotationSchedule: []timeSchedule{
					{},
				},
				directory: os.TempDir(),
				fileBase:  "logfeller.test-logfeller",
				ext:       ".log",
			},
		},
		{
			name: "sort_schedules_offsets",
			f: &File{
				When:             "Y",
				RotationSchedule: []string{"1202 2311:55", "0102 0821:22", "0102 0821:22", "0109 1504:05", "0102 0504:05", "0102 0544:05", "0102 0544:32", "0611 1504:05"},
			},
			want: wantFields{
				Filename:         filepath.Join(os.TempDir(), "logfeller.test-logfeller.log"),
				When:             "y",
				RotationSchedule: []string{"1202 2311:55", "0102 0821:22", "0102 0821:22", "0109 1504:05", "0102 0504:05", "0102 0544:05", "0102 0544:32", "0611 1504:05"},
				BackupTimeFormat: ".2006-01-02T1504-05",
				timeRotationSchedule: []timeSchedule{
					{month: 1, day: 2, hour: 5, minute: 04, second: 5},
					{month: 1, day: 2, hour: 5, minute: 44, second: 5},
					{month: 1, day: 2, hour: 5, minute: 44, second: 32},
					{month: 1, day: 2, hour: 8, minute: 21, second: 22},
					{month: 1, day: 2, hour: 8, minute: 21, second: 22},
					{month: 1, day: 9, hour: 15, minute: 04, second: 5},
					{month: 6, day: 11, hour: 15, minute: 04, second: 5},
					{month: 12, day: 2, hour: 23, minute: 11, second: 55},
				},
				directory: os.TempDir(),
				fileBase:  "logfeller.test-logfeller",
				ext:       ".log",
			},
		},
		{
			name: "schedule_parsing_error",
			f: &File{
				When:             "Y",
				RotationSchedule: []string{"1302 231155"},
			},
			wantErr: true,
		},
		{
			name:    "When_invalid_error",
			f:       &File{When: "HOUR"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.f.init()
			testutils.TrueOrFatal(t, (err != nil) == tt.wantErr, "File.init() error = %v, wantErr %v", err, tt.wantErr)
			if err != nil {
				return
			}
			testutils.TrueOrError(t, tt.f.Filename == tt.want.Filename, "File.Filename = %v, want %v", tt.f.Filename, tt.want.Filename)
			testutils.TrueOrError(t, tt.f.When == tt.want.When, "File.When = %v, want %v", tt.f.When, tt.want.When)
			testutils.TrueOrError(t, reflect.DeepEqual(tt.f.RotationSchedule, tt.want.RotationSchedule), "File.RotationSchedule = %v, want %v", tt.f.RotationSchedule, tt.want.RotationSchedule)
			testutils.TrueOrError(t, tt.f.UseLocal == tt.want.UseLocal, "File.UseLocal = %v, want %v", tt.f.UseLocal, tt.want.UseLocal)
			testutils.TrueOrError(t, tt.f.Backups == tt.want.Backups, "File.Backups = %v, want %v", tt.f.Backups, tt.want.Backups)
			testutils.TrueOrError(t, reflect.DeepEqual(tt.f.timeRotationSchedule, tt.want.timeRotationSchedule), "File.timeRotationSchedule = %v, want %v", tt.f.timeRotationSchedule, tt.want.timeRotationSchedule)
			testutils.TrueOrError(t, tt.f.BackupTimeFormat == tt.want.BackupTimeFormat, "File.BackupTimeFormat = %v, want %v", tt.f.BackupTimeFormat, tt.want.BackupTimeFormat)
			testutils.TrueOrError(t, tt.f.directory == tt.want.directory, "File.directory = %v, want %v", tt.f.directory, tt.want.directory)
			testutils.TrueOrError(t, tt.f.fileBase == tt.want.fileBase, "File.fileBase = %v, want %v", tt.f.fileBase, tt.want.fileBase)
			testutils.TrueOrError(t, tt.f.ext == tt.want.ext, "File.ext = %v, want %v", tt.f.ext, tt.want.ext)
		})
	}
}

// TestFile_UnmarshalJSON is purely there to see if mapping between JSON tag
// fields are accurate. For the actual init check TestFile_init
func TestFile_UnmarshalJSON(t *testing.T) {
	type args struct {
		data []byte
	}
	type wantFields struct {
		Filename         string
		When             WhenRotate
		RotationSchedule []string
		UseLocal         bool
		Backups          int
		BackupTimeFormat string
	}
	tests := []struct {
		name    string
		args    args
		want    wantFields
		wantErr bool
	}{
		{
			name: "json_mapped_properly",
			args: args{
				data: []byte(`{
	"filename":         "some-file-proper-name.txt",
	"when":             "M",
	"rotation_schedule": ["03 1430:00", "10 1200:00"],
	"use_local":         true,
	"backups":          69,
	"backup_time_format": "Jan _2 15:04:05"
}`),
			},
			want: wantFields{
				Filename:         "some-file-proper-name.txt",
				When:             "m",
				RotationSchedule: []string{"03 1430:00", "10 1200:00"},
				UseLocal:         true,
				Backups:          69,
				BackupTimeFormat: "Jan _2 15:04:05",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f File
			err := json.Unmarshal(tt.args.data, &f)
			testutils.TrueOrFatal(t, (err != nil) == tt.wantErr, "File.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			if err != nil {
				return
			}
			testutils.TrueOrError(t, f.Filename == tt.want.Filename, "File.Filename = %v, want %v", f.Filename, tt.want.Filename)
			testutils.TrueOrError(t, f.When == tt.want.When, "File.When = %v, want %v", f.When, tt.want.When)
			testutils.TrueOrError(t, reflect.DeepEqual(f.RotationSchedule, tt.want.RotationSchedule), "File.RotationSchedule = %v, want %v", f.RotationSchedule, tt.want.RotationSchedule)
			testutils.TrueOrError(t, f.UseLocal == tt.want.UseLocal, "File.UseLocal = %v, want %v", f.UseLocal, tt.want.UseLocal)
			testutils.TrueOrError(t, f.Backups == tt.want.Backups, "File.Backups = %v, want %v", f.Backups, tt.want.Backups)
		})
	}
}

// TestFile_UnmarshalYAML is purely there to see if mapping between JSON tag
// fields are accurate. For the actual init check TestFile_init
func TestFile_UnmarshalYAML(t *testing.T) {
	type args struct {
		data []byte
	}
	type wantFields struct {
		Filename         string
		When             WhenRotate
		RotationSchedule []string
		UseLocal         bool
		Backups          int
		BackupTimeFormat string
	}
	tests := []struct {
		name    string
		args    args
		want    wantFields
		wantErr bool
	}{
		{
			name: "yaml_mapped_properly",
			args: args{
				data: []byte(`
filename: some-file-proper-name.txt
when: m
rotation-schedule: ["03 1430:00", "10 1200:00"]
use-local: true
backups: 69
backup-time-format: "Jan _2 15:04:05"`),
			},
			want: wantFields{
				Filename:         "some-file-proper-name.txt",
				When:             "m",
				RotationSchedule: []string{"03 1430:00", "10 1200:00"},
				UseLocal:         true,
				Backups:          69,
				BackupTimeFormat: "Jan _2 15:04:05",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f File
			err := yaml.Unmarshal(tt.args.data, &f)
			testutils.TrueOrFatal(t, (err != nil) == tt.wantErr, "File.UnmarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
			testutils.TrueOrError(t, f.Filename == tt.want.Filename, "File.Filename = %v, want %v", f.Filename, tt.want.Filename)
			testutils.TrueOrError(t, f.When == tt.want.When, "File.When = %v, want %v", f.When, tt.want.When)
			testutils.TrueOrError(t, reflect.DeepEqual(f.RotationSchedule, tt.want.RotationSchedule), "File.RotationSchedule = %v, want %v", f.RotationSchedule, tt.want.RotationSchedule)
			testutils.TrueOrError(t, f.UseLocal == tt.want.UseLocal, "File.UseLocal = %v, want %v", f.UseLocal, tt.want.UseLocal)
			testutils.TrueOrError(t, f.Backups == tt.want.Backups, "File.Backups = %v, want %v", f.Backups, tt.want.Backups)

		})
	}
}

// TestFile contains a set of generic test cases dealing with how rotational logic
// and writing works with logfeller.
func TestFile(t *testing.T) {
	fname := "foo.log"
	tests := []struct {
		name string
		// the actual test to act on, should return the filenames and expected
		// contents that were written by the the rotational file handler.
		// This is also where we run can other assertions that arent generic.
		do func(t testing.TB, dirname string) map[string][]byte
	}{
		{
			name: "write_new_file",
			do: func(t testing.TB, dirname string) map[string][]byte {
				rf := File{Filename: filepath.Join(dirname, fname)}
				b := []byte("BARBAR")
				n, err := rf.Write(b)
				defer rf.Close()
				testutils.TrueOrFatal(t, err == nil, "write error; filename=%s;err=%v", fname, err)
				testutils.TrueOrFatal(t, n == len(b), "write length mismatch; filename=%s;n=%d;datalen=%d", fname, n, len(b))
				return map[string][]byte{fname: b}
			},
		},
		{
			name: "write_to_existing_file",
			do: func(t testing.TB, dirname string) map[string][]byte {
				fullpath := filepath.Join(dirname, fname)
				// write an existing file
				err := ioutil.WriteFile(fullpath, []byte("BARBAREXISTING\n"), 0600)
				testutils.TrueOrFatal(t, err == nil, "write existing file error; filename=%s;err=%v", fname, err)

				rf := File{Filename: fullpath}
				defer rf.Close()
				b := []byte("BARBAR2")
				n, err := rf.Write(b)
				testutils.TrueOrFatal(t, err == nil, "write error; filename=%s;err=%v", fname, err)
				testutils.TrueOrFatal(t, n == len(b), "write length mismatch; filename=%s;n=%d;datalen=%d", fname, n, len(b))
				return map[string][]byte{fname: []byte("BARBAREXISTING\nBARBAR2")}
			},
		},
		{
			name: "write_multiple_times_within_the_hour_to_existing_file_no_rotate",
			do: func(t testing.TB, dirname string) map[string][]byte {
				staticTime := time.Date(2020, 8, 9, 10, 0, 0, 0, time.Local)
				var mockNow = func() time.Time { return staticTime }
				var mockNow20minsAfter = func() time.Time { return staticTime.Add(20 * time.Minute) }
				var mockNow40minsAfter = func() time.Time { return staticTime.Add(40 * time.Minute) }
				fullpath := filepath.Join(dirname, fname)
				// write an existing file
				err := ioutil.WriteFile(fullpath, []byte("BARBAREXISTING\n"), 0600)
				testutils.TrueOrFatal(t, err == nil, "write existing file error; filename=%s;err=%v", fname, err)

				rf := File{Filename: fullpath, When: "H", nowFunc: mockNow}
				defer rf.Close()
				b := []byte("BARBAR2\n")
				n, err := rf.Write(b)
				testutils.TrueOrFatal(t, err == nil, "write error; filename=%s;err=%v", fname, err)
				testutils.TrueOrFatal(t, n == len(b), "write length mismatch; filename=%s;n=%d;datalen=%d", fname, n, len(b))

				rf.setNowFunc(mockNow20minsAfter)
				b = []byte("BARBAR3\n")
				n, err = rf.Write(b)
				testutils.TrueOrFatal(t, err == nil, "write error; filename=%s;err=%v", fname, err)
				testutils.TrueOrFatal(t, n == len(b), "write length mismatch; filename=%s;n=%d;datalen=%d", fname, n, len(b))

				rf.setNowFunc(mockNow40minsAfter)
				b = []byte("BARBAR4\n")
				n, err = rf.Write(b)
				testutils.TrueOrFatal(t, err == nil, "write error; filename=%s;err=%v", fname, err)
				testutils.TrueOrFatal(t, n == len(b), "write length mismatch; filename=%s;n=%d;datalen=%d", fname, n, len(b))

				return map[string][]byte{fname: []byte("BARBAREXISTING\nBARBAR2\nBARBAR3\nBARBAR4\n")}
			},
		},
		{
			name: "rotate_immedately_before_write",
			do: func(t testing.TB, dirname string) map[string][]byte {
				now := time.Now()
				oneDayLater := now.Add(24 * time.Hour)
				var mock1DayLater = func() time.Time { return oneDayLater }
				fullpath := filepath.Join(dirname, fname)
				// write an existing file
				err := ioutil.WriteFile(fullpath, []byte("BARBAREXISTING\n"), 0600)
				testutils.TrueOrFatal(t, err == nil, "write existing file error; filename=%s;err=%v", fname, err)

				// Force rotation by mocking now to return now 1 day later
				rf := File{Filename: fullpath, nowFunc: mock1DayLater}
				defer rf.Close()
				b := []byte("BARBAR2\n")
				n, err := rf.Write(b)
				testutils.TrueOrFatal(t, err == nil, "write error; filename=%s;err=%v", fname, err)
				testutils.TrueOrFatal(t, n == len(b), "write length mismatch; filename=%s;n=%d;datalen=%d", fname, n, len(b))

				rotatedFilename := fmt.Sprint("foo", testutils.TimeOfDay(time.Now(), 0, 0, 0).Format(defaultBackupTimeFormat), ".log")
				return map[string][]byte{
					fname:           []byte("BARBAR2\n"),
					rotatedFilename: []byte("BARBAREXISTING\n"),
				}
			},
		},
		{
			name: "rotate_daily_with_multiple_schedules",
			do: func(t testing.TB, dirname string) map[string][]byte {
				startOfDay := testutils.TimeOfDay(time.Now(), 0, 0, 0)
				fullpath := filepath.Join(dirname, fname)
				yesterday1600 := startOfDay.Add(-8 * time.Hour)

				b1 := []byte("BARBAREXISTING\n")
				// write an existing file, mock it to say last edited was yesterday 4pm
				err := ioutil.WriteFile(fullpath, b1, 0600)
				testutils.TrueOrFatal(t, err == nil, "write existing file error; filename=%s;err=%v", fname, err)
				err = os.Chtimes(fullpath, yesterday1600, yesterday1600)
				testutils.TrueOrFatal(t, err == nil, "should not have error changing modified times; filename=%s;err=%v", fname, err)

				// Force rotation by mocking now to return now 1 day later
				rf := File{
					Filename:         fullpath,
					When:             "d",
					RotationSchedule: []string{"0100:00", "0830:00", "1400:00", "1900:00"},
					nowFunc:          func() time.Time { return startOfDay },
					UseLocal:         true,
				}
				defer rf.Close()

				// First rotation, file was created at 1600, so rotation time will be 1400
				firstRotateFilename := fmt.Sprint("foo", testutils.TimeOfDay(yesterday1600, 14, 0, 0).Format(defaultBackupTimeFormat), ".log")
				b2 := []byte("BARBAR2\n")
				n, err := rf.Write(b2)
				testutils.TrueOrFatal(t, err == nil, "write error b2 err: content=%s,err=%v", b2, err)
				testutils.TrueOrFatal(t, n == len(b2), "write b2 length mismatch; n=%d, expected=%d", n, len(b2))
				// Another write but within the same timeslot (13mins from start of day)
				rf.setNowFunc(func() time.Time { return startOfDay.Add(13 * time.Minute) })
				b3 := []byte("BARBAR3\n")
				n, err = rf.Write(b3)
				testutils.TrueOrFatal(t, err == nil, "write error b3 err: content=%s,err=%v", b3, err)
				testutils.TrueOrFatal(t, n == len(b3), "write b3 length mismatch; n=%d, expected=%d", n, len(b3))
				// Another write but within the same timeslot (49mins from start of day)
				rf.setNowFunc(func() time.Time { return startOfDay.Add(49 * time.Minute) })
				b4 := []byte("BARBAR4\n")
				n, err = rf.Write(b4)
				testutils.TrueOrFatal(t, err == nil, "write error b4 err: content=%s,err=%v", b4, err)
				testutils.TrueOrFatal(t, n == len(b4), "write b4 length mismatch; n=%d, expected=%d", n, len(b4))

				// Second rotation, write at 0105am, rotote and filename should be at 7pm
				secondRotateFilename := fmt.Sprint("foo", testutils.TimeOfDay(yesterday1600, 19, 0, 0).Format(defaultBackupTimeFormat), ".log")
				rf.setNowFunc(func() time.Time { return startOfDay.Add(65 * time.Minute) })
				b5 := []byte("BARBAR5\n")
				n, err = rf.Write(b5)
				testutils.TrueOrFatal(t, err == nil, "write error b5 err: content=%s,err=%v", b5, err)
				testutils.TrueOrFatal(t, n == len(b5), "write b5 length mismatch; n=%d, expected=%d", n, len(b5))

				// Third rotation, write at 9am, rotate and filename should be at 1am
				thirdRotateFilename := fmt.Sprint("foo", testutils.TimeOfDay(startOfDay, 1, 0, 0).Format(defaultBackupTimeFormat), ".log")
				rf.setNowFunc(func() time.Time { return startOfDay.Add(9 * time.Hour) })
				b6 := []byte("BARBAR6\n")
				n, err = rf.Write(b6)
				testutils.TrueOrFatal(t, err == nil, "write error b6 err: content=%s,err=%v", b6, err)
				testutils.TrueOrFatal(t, n == len(b6), "write b6 length mismatch; n=%d, expected=%d", n, len(b6))
				// another write, no rotate at 11am
				rf.setNowFunc(func() time.Time { return startOfDay.Add(11 * time.Hour) })
				b7 := []byte("BARBAR7\n")
				n, err = rf.Write(b7)
				testutils.TrueOrFatal(t, err == nil, "write error b7 err: content=%s,err=%v", b7, err)
				testutils.TrueOrFatal(t, n == len(b7), "write b7 length mismatch; n=%d, expected=%d", n, len(b7))

				// Fourth rotation, write at 3pm, rotate and filename should be at 8.30am
				fourthRotateFilename := fmt.Sprint("foo", testutils.TimeOfDay(startOfDay, 8, 30, 0).Format(defaultBackupTimeFormat), ".log")
				rf.setNowFunc(func() time.Time { return startOfDay.Add(15 * time.Hour) })
				b8 := []byte("BARBAR8\n")
				n, err = rf.Write(b8)
				testutils.TrueOrFatal(t, err == nil, "write error b8 err: content=%s,err=%v", b8, err)
				testutils.TrueOrFatal(t, n == len(b8), "write b8 length mismatch; n=%d, expected=%d", n, len(b8))
				// another write, no rotate at 5pm
				rf.setNowFunc(func() time.Time { return startOfDay.Add(17 * time.Hour) })
				b9 := []byte("BARBAR9\n")
				n, err = rf.Write(b9)
				testutils.TrueOrFatal(t, err == nil, "write error b9 err: content=%s,err=%v", b9, err)
				testutils.TrueOrFatal(t, n == len(b9), "write b9 length mismatch; n=%d, expected=%d", n, len(b9))

				// Fifth rotation, write at 8pm, rotate and filename should be at 2pm
				fifthRotateFilename := fmt.Sprint("foo", testutils.TimeOfDay(startOfDay, 14, 0, 0).Format(defaultBackupTimeFormat), ".log")
				rf.setNowFunc(func() time.Time { return startOfDay.Add(20 * time.Hour) })
				b10 := []byte("BARBAR10\n")
				n, err = rf.Write(b10)
				testutils.TrueOrFatal(t, err == nil, "write error b10 err: content=%s,err=%v", b10, err)
				testutils.TrueOrFatal(t, n == len(b10), "write b10 length mismatch; n=%d, expected=%d", n, len(b10))
				// another write, no rotate at 11pm
				rf.setNowFunc(func() time.Time { return startOfDay.Add(23 * time.Hour) })
				b11 := []byte("BARBAR11\n")
				n, err = rf.Write(b11)
				testutils.TrueOrFatal(t, err == nil, "write error b11 err: content=%s,err=%v", b11, err)
				testutils.TrueOrFatal(t, n == len(b11), "write b11 length mismatch; n=%d, expected=%d", n, len(b11))

				// Sixth rotation, write at 2am, rotate and filename should be at 7pm
				sixthRotateFilename := fmt.Sprint("foo", testutils.TimeOfDay(startOfDay, 19, 0, 0).Format(defaultBackupTimeFormat), ".log")
				rf.setNowFunc(func() time.Time { return startOfDay.Add(26 * time.Hour) })
				b12 := []byte("BARBAR12\n")
				n, err = rf.Write(b12)
				testutils.TrueOrFatal(t, err == nil, "write error b12 err: content=%s,err=%v", b12, err)
				testutils.TrueOrFatal(t, n == len(b12), "write b12 length mismatch; n=%d, expected=%d", n, len(b12))
				// another write, no rotate at 4am
				rf.setNowFunc(func() time.Time { return startOfDay.Add(28 * time.Hour) })
				b13 := []byte("BARBAR13\n")
				n, err = rf.Write(b13)
				testutils.TrueOrFatal(t, err == nil, "write error b13 err: content=%s,err=%v", b13, err)
				testutils.TrueOrFatal(t, n == len(b13), "write b13 length mismatch; n=%d, expected=%d", n, len(b13))

				return map[string][]byte{
					firstRotateFilename:  []byte("BARBAREXISTING\n"),
					secondRotateFilename: []byte("BARBAR2\nBARBAR3\nBARBAR4\n"),
					thirdRotateFilename:  []byte("BARBAR5\n"),
					fourthRotateFilename: []byte("BARBAR6\nBARBAR7\n"),
					fifthRotateFilename:  []byte("BARBAR8\nBARBAR9\n"),
					sixthRotateFilename:  []byte("BARBAR10\nBARBAR11\n"),
					fname:                []byte("BARBAR12\nBARBAR13\n"),
				}
			},
		},
		{
			name: "rotate_backup_clear_file_of_no_clear_diff_format",
			do: func(t testing.TB, dirname string) map[string][]byte {
				now := time.Now()
				backupFormat := ".2006-01-02-1504"

				// Initial data, will be deleted eventually
				fullpath := filepath.Join(dirname, fname)
				err := ioutil.WriteFile(fullpath, []byte("BARBAREXISTING\n"), 0600)
				testutils.TrueOrFatal(t, err == nil, "write existing file error; filename=%s;err=%v", fname, err)

				// extra files (shouldnt be deleted)
				// Looks similar to one expected but this file is foo.2006-01-02T1504-05.log which is diff from the
				// backup time format we are using. This file shouldnt be touched at all.
				extraFilename1 := fmt.Sprint("foo", testutils.TimeOfDay(now.Add(24*time.Hour), 0, 0, 0).Format(defaultBackupTimeFormat), ".log")
				err = ioutil.WriteFile(filepath.Join(dirname, extraFilename1), []byte("FOO_EXTRA1\n"), 0600)
				testutils.TrueOrFatal(t, err == nil, "write existing file error; filename=%s;err=%v", fname, err)
				// Different file name from original log file name.
				extraFilename2 := "bar.log"
				err = ioutil.WriteFile(filepath.Join(dirname, extraFilename2), []byte("BAR_EXTRA1\n"), 0600)
				testutils.TrueOrFatal(t, err == nil, "write existing file error; filename=%s;err=%v", fname, err)

				rf := File{Filename: fullpath, Backups: 1, BackupTimeFormat: backupFormat}
				defer rf.Close()

				// Force rotation by mocking now to return now 1 day later
				rf.setNowFunc(func() time.Time { return now.Add(24 * time.Hour) })
				b1 := []byte("BARBAR2\n")
				n, err := rf.Write(b1)
				testutils.TrueOrFatal(t, err == nil, "write error b1 err: content=%s,err=%v", b1, err)
				testutils.TrueOrFatal(t, n == len(b1), "write b1 length mismatch; n=%d, expected=%d", n, len(b1))

				// Force another rotation by mocking now to return now 2 day later
				firstRotateFilename := fmt.Sprint("foo", testutils.TimeOfDay(now.Add(24*time.Hour), 0, 0, 0).Format(backupFormat), ".log")
				rf.setNowFunc(func() time.Time { return now.Add(48 * time.Hour) })
				b2 := []byte("BARBAR3\n")
				n, err = rf.Write(b2)
				testutils.TrueOrFatal(t, err == nil, "write error b2 err: content=%s,err=%v", b2, err)
				testutils.TrueOrFatal(t, n == len(b2), "write b2 length mismatch; n=%d, expected=%d", n, len(b2))
				return map[string][]byte{
					fname:               []byte("BARBAR3\n"),
					firstRotateFilename: []byte("BARBAR2\n"),
					extraFilename1:      []byte("FOO_EXTRA1\n"),
					extraFilename2:      []byte("BAR_EXTRA1\n"),
				}
			},
		},
		{
			name: "force_rotate_flush_file",
			do: func(t testing.TB, dirname string) map[string][]byte {
				now := time.Now()
				startOfDay := testutils.TimeOfDay(now, 0, 0, 0)
				fullpath := filepath.Join(dirname, fname)

				rf := File{Filename: fullpath, Backups: 2}
				defer rf.Close()

				b1 := []byte("BARBAR1\n")
				n, err := rf.Write(b1)
				testutils.TrueOrFatal(t, err == nil, "write error b1 err: content=%s,err=%v", b1, err)
				testutils.TrueOrFatal(t, n == len(b1), "write b1 length mismatch; n=%d, expected=%d", n, len(b1))
				err = rf.Rotate()
				testutils.TrueOrFatal(t, err == nil, "rotate b1 err: err=%v", err)

				b2 := []byte("BARBAR2\n")
				n, err = rf.Write(b2)
				testutils.TrueOrFatal(t, err == nil, "write error b2 err: content=%s,err=%v", b2, err)
				testutils.TrueOrFatal(t, n == len(b2), "write b2 length mismatch; n=%d, expected=%d", n, len(b2))
				err = rf.Rotate()
				testutils.TrueOrFatal(t, err == nil, "rotate b2 err: err=%v", err)

				b3 := []byte("BARBAR3\n")
				n, err = rf.Write(b3)
				testutils.TrueOrFatal(t, err == nil, "write error b3 err: content=%s,err=%v", b3, err)
				testutils.TrueOrFatal(t, n == len(b3), "write b3 length mismatch; n=%d, expected=%d", n, len(b3))
				err = rf.Rotate()
				testutils.TrueOrFatal(t, err == nil, "rotate b3 err: err=%v", err)

				// Try to write again rotating via normal means
				// This will not rotate because we are "rotating" an empty file
				// The logic prevents that and just reuses the empty file without
				// moving
				rf.setNowFunc(func() time.Time { return now.Add(24 * time.Hour) })
				b4 := []byte("BARBAR4\n")
				n, err = rf.Write(b4)
				testutils.TrueOrFatal(t, err == nil, "write error b4 err: content=%s,err=%v", b4, err)
				testutils.TrueOrFatal(t, n == len(b4), "write b4 length mismatch; n=%d, expected=%d", n, len(b4))

				firstRotateFilename := fmt.Sprint("foo", startOfDay.Format(defaultBackupTimeFormat), ".log")
				return map[string][]byte{
					firstRotateFilename: []byte("BARBAR1\nBARBAR2\nBARBAR3\n"),
					fname:               []byte("BARBAR4\n"),
				}
			},
		},
		{
			name: "clear_previous_backup_before_writing",
			do: func(t testing.TB, dirname string) map[string][]byte {
				now := time.Now()
				startOfYesterday := testutils.TimeOfDay(now.Add(-24*time.Hour), 0, 0, 0)
				startOf2DaysBefore := testutils.TimeOfDay(now.Add(-48*time.Hour), 0, 0, 0)
				fullpath := filepath.Join(dirname, fname)

				// Multiple backup files, only backup file 2 will be remain
				backupFilename1 := fmt.Sprint("foo", startOf2DaysBefore.Format(defaultBackupTimeFormat), ".log")
				err := ioutil.WriteFile(filepath.Join(dirname, backupFilename1), []byte("BARBAREXISTING_1\n"), 0600)
				testutils.TrueOrFatal(t, err == nil, "write existing file error; filename=%s;err=%v", fname, err)
				backupFilename2 := fmt.Sprint("foo", startOfYesterday.Format(defaultBackupTimeFormat), ".log")
				err = ioutil.WriteFile(filepath.Join(dirname, backupFilename2), []byte("BARBAREXISTING_2\n"), 0600)
				testutils.TrueOrFatal(t, err == nil, "write existing file error; filename=%s;err=%v", fname, err)

				rf := File{Filename: fullpath, Backups: 1}
				defer rf.Close()

				b1 := []byte("BARBAR1\n")
				n, err := rf.Write(b1)
				testutils.TrueOrFatal(t, err == nil, "write error b1 err: content=%s,err=%v", b1, err)
				testutils.TrueOrFatal(t, n == len(b1), "write b1 length mismatch; n=%d, expected=%d", n, len(b1))

				return map[string][]byte{
					backupFilename2: []byte("BARBAREXISTING_2\n"),
					fname:           []byte("BARBAR1\n"),
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dirname, err := testutils.MkTestDir(tt.name)
			testutils.TrueOrFatal(t, err == nil, "should not fail at creating test dir; dir=%s, error = %v", dirname, err)
			defer func() {
				errInner := os.RemoveAll(dirname)
				testutils.TrueOrFatal(t, errInner == nil, "failed to cleanup test folder; dir=%s, err=%v", dirname, errInner)
			}()

			expectedFilenamesAndContents := tt.do(t, dirname)
			dirEntries, err := os.ReadDir(dirname)
			testutils.TrueOrFatal(t, err == nil, "should not fail at reading dir entries; dirname=%s,err=%v", dirname, err)
			for _, de := range dirEntries {
				if de.IsDir() {
					t.Logf("file entry %s is a directory, skipping", de.Name())
					continue
				}
				expectContent, ok := expectedFilenamesAndContents[de.Name()]
				if testutils.TrueOrError(t, ok, "filename written is not expected; filename=%s", de.Name()) {
					continue
				}
				delete(expectedFilenamesAndContents, de.Name())
				fullpath := filepath.Join(dirname, de.Name())
				fcontent, err := ioutil.ReadFile(fullpath)
				if testutils.TrueOrError(t, err == nil, "should not fail reading written file content; filename=%s, err=%v", de.Name(), err) {
					continue
				}
				testutils.TrueOrError(t, reflect.DeepEqual(fcontent, expectContent),
					"filecontent should match; file=%s, filecontent=%s, wantcontent=%s", de.Name(), fcontent, expectContent,
				)
			}
			// here if still have entries, its considered a failed test as
			// the expected content doesnt exist
			for wantFilename, wantContent := range expectedFilenamesAndContents {
				t.Errorf("want files still exist that don't exist in directory;wantFilename=%s,wantContent=%s", wantFilename, wantContent)
			}
		})
	}
}
