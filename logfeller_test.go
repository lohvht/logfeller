// package logfeller implements a library for writing to and rotating files
// based on a timed schedule.
package logfeller

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
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
			name: "omit_most_fields",
			f:    &File{When: "D"},
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
			if (err != nil) != tt.wantErr {
				t.Errorf("File.init() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if tt.f.Filename != tt.want.Filename {
				t.Errorf("File.Filename = %v, want %v", tt.f.Filename, tt.want.Filename)
			}
			if tt.f.When != tt.want.When {
				t.Errorf("File.When = %v, want %v", tt.f.When, tt.want.When)
			}
			if !reflect.DeepEqual(tt.f.RotationSchedule, tt.want.RotationSchedule) {
				t.Errorf("File.RotationSchedule = %v, want %v", tt.f.RotationSchedule, tt.want.RotationSchedule)
			}
			if tt.f.UseLocal != tt.want.UseLocal {
				t.Errorf("File.UseLocal = %v, want %v", tt.f.UseLocal, tt.want.UseLocal)
			}
			if tt.f.Backups != tt.want.Backups {
				t.Errorf("File.Backups = %v, want %v", tt.f.Backups, tt.want.Backups)
			}
			if !reflect.DeepEqual(tt.f.timeRotationSchedule, tt.want.timeRotationSchedule) {
				t.Errorf("File.timeRotationSchedule = %v, want %v", tt.f.timeRotationSchedule, tt.want.timeRotationSchedule)
			}
			if tt.f.BackupTimeFormat != tt.want.BackupTimeFormat {
				t.Errorf("File.BackupTimeFormat = %v, want %v", tt.f.BackupTimeFormat, tt.want.BackupTimeFormat)
			}
			if tt.f.directory != tt.want.directory {
				t.Errorf("File.directory = %v, want %v", tt.f.directory, tt.want.directory)
			}
			if tt.f.fileBase != tt.want.fileBase {
				t.Errorf("File.fileBase = %v, want %v", tt.f.fileBase, tt.want.fileBase)
			}
			if tt.f.ext != tt.want.ext {
				t.Errorf("File.ext = %v, want %v", tt.f.ext, tt.want.ext)
			}
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
			if (err != nil) != tt.wantErr {
				t.Errorf("File.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if f.Filename != tt.want.Filename {
				t.Errorf("File.Filename = %v, want %v", f.Filename, tt.want.Filename)
			}
			if f.When != tt.want.When {
				t.Errorf("File.When = %v, want %v", f.When, tt.want.When)
			}
			if !reflect.DeepEqual(f.RotationSchedule, tt.want.RotationSchedule) {
				t.Errorf("File.RotationSchedule = %v, want %v", f.RotationSchedule, tt.want.RotationSchedule)
			}
			if f.UseLocal != tt.want.UseLocal {
				t.Errorf("File.UseLocal = %v, want %v", f.UseLocal, tt.want.UseLocal)
			}
			if f.Backups != tt.want.Backups {
				t.Errorf("File.Backups = %v, want %v", f.Backups, tt.want.Backups)
			}
		})
	}
}
