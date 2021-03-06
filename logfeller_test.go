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
		timeRotationSchedule []timeOffset
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
				RotationSchedule: []string{"1430", "1200"},
				UseLocal:         true,
				Backups:          40,
				BackupTimeFormat: "Jan _2 15:04:05",
			},
			want: wantFields{
				Filename:         "file.txt",
				When:             "h",
				RotationSchedule: []string{"1430", "1200"},
				UseLocal:         true,
				Backups:          40,
				BackupTimeFormat: "Jan _2 15:04:05",
				timeRotationSchedule: []timeOffset{
					{minute: 12}, {minute: 14, second: 30},
				},
			},
		},
		{
			name: "omit_most_fields",
			f:    &File{When: "D"},
			want: wantFields{
				Filename:             filepath.Join(os.TempDir(), "logfeller.test-logfeller.log"),
				When:                 "d",
				BackupTimeFormat:     "2006-01-02.150405",
				timeRotationSchedule: []timeOffset{{}},
			},
		},
		{
			name: "sort_schedules_offsets",
			f: &File{
				When:             "Y",
				RotationSchedule: []string{"1202 231155", "0102 082122", "0102 082122", "0109 150405", "0102 050405", "0102 054405", "0102 054432", "0611 150405"},
			},
			want: wantFields{
				Filename:         filepath.Join(os.TempDir(), "logfeller.test-logfeller.log"),
				When:             "y",
				RotationSchedule: []string{"1202 231155", "0102 082122", "0102 082122", "0109 150405", "0102 050405", "0102 054405", "0102 054432", "0611 150405"},
				BackupTimeFormat: "2006-01-02.150405",
				timeRotationSchedule: []timeOffset{
					{month: 1, day: 2, hour: 5, minute: 04, second: 5},
					{month: 1, day: 2, hour: 8, minute: 21, second: 22},
					{month: 1, day: 2, hour: 8, minute: 21, second: 22},
					{month: 1, day: 2, hour: 5, minute: 44, second: 5},
					{month: 1, day: 2, hour: 5, minute: 44, second: 32},
					{month: 1, day: 9, hour: 15, minute: 04, second: 5},
					{month: 6, day: 1, hour: 15, minute: 04, second: 5},
					{month: 12, day: 2, hour: 23, minute: 11, second: 55},
				},
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
			if tt.f.BackupTimeFormat != tt.want.BackupTimeFormat {
				t.Errorf("File.BackupTimeFormat = %v, want %v", tt.f.BackupTimeFormat, tt.want.BackupTimeFormat)
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
	"rotation_schedule": ["03 143000", "10 120000"],
	"use_local":         true,
	"backups":          69,
	"backup_time_format": "Jan _2 15:04:05"
}`),
			},
			want: wantFields{
				Filename:         "some-file-proper-name.txt",
				When:             "m",
				RotationSchedule: []string{"03 143000", "10 120000"},
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

func TestWhenRotate_valid(t *testing.T) {
	tests := []struct {
		name    string
		r       WhenRotate
		wantErr bool
	}{
		{name: "hourly_lower", r: "h"},
		{name: "hourly_upper", r: "H"},
		{name: "daily_lower", r: "d"},
		{name: "daily_upper", r: "D"},
		{name: "monthly_lower", r: "m"},
		{name: "monthly_upper", r: "M"},
		{name: "yearly_lower", r: "y"},
		{name: "yearly_upper", r: "Y"},
		{name: "invalid_singlechar", r: "A", wantErr: true},
		{name: "invalid_multiplechar", r: "HOUR", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.r.valid(); (err != nil) != tt.wantErr {
				t.Errorf("WhenRotate.valid() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWhenRotate_baseRotateTime(t *testing.T) {
	tests := []struct {
		name string
		r    WhenRotate
		want timeOffset
	}{
		{name: "hourly_lower", r: "h", want: timeOffset{}},
		{name: "hourly_upper", r: "H", want: timeOffset{}},
		{name: "daily_lower", r: "d", want: timeOffset{}},
		{name: "daily_upper", r: "D", want: timeOffset{}},
		{name: "monthly_lower", r: "m", want: timeOffset{day: 1}},
		{name: "monthly_upper", r: "M", want: timeOffset{day: 1}},
		{name: "yearly_lower", r: "y", want: timeOffset{day: 1, month: 1}},
		{name: "yearly_upper", r: "Y", want: timeOffset{day: 1, month: 1}},
		{name: "invalid_singlechar", r: "A", want: timeOffset{day: 1, month: 1}},
		{name: "invalid_multiplechar", r: "HOUR", want: timeOffset{day: 1, month: 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.baseRotateTime(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WhenRotate.baseRotateTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWhenRotate_parseTimeoffset(t *testing.T) {
	type args struct {
		offsetStr string
	}
	tests := []struct {
		name    string
		r       WhenRotate
		args    args
		want    timeOffset
		wantErr bool
	}{
		{name: "hourly", r: "H", args: args{offsetStr: "1445"}, want: timeOffset{minute: 14, second: 45}},
		{name: "daily", r: "d", args: args{offsetStr: "191445"}, want: timeOffset{hour: 19, minute: 14, second: 45}},
		{name: "monthly", r: "M", args: args{offsetStr: "15 191445"}, want: timeOffset{day: 15, hour: 19, minute: 14, second: 45}},
		{name: "yearly", r: "y", args: args{offsetStr: "0615 191445"}, want: timeOffset{month: 6, day: 15, hour: 19, minute: 14, second: 45}},
		{name: "when_error", r: "HOUR", wantErr: true},
		{name: "hourly_format_invalid", r: "h", args: args{offsetStr: "114451"}, wantErr: true},
		{name: "daily_format_invalid", r: "D", args: args{offsetStr: "1 114451"}, wantErr: true},
		{name: "monthly_format_invalid", r: "m", args: args{offsetStr: "111 114451"}, wantErr: true},
		{name: "yearly_format_invalid", r: "Y", args: args{offsetStr: "31111 114451"}, wantErr: true},
		{name: "second_exeed", r: "y", args: args{offsetStr: "0615 190061"}, wantErr: true},
		{name: "minute_exeed", r: "y", args: args{offsetStr: "0615 196159"}, wantErr: true},
		{name: "hour_exeed", r: "y", args: args{offsetStr: "0615 245959"}, wantErr: true},
		{name: "day_exeed", r: "y", args: args{offsetStr: "0632 245959"}, wantErr: true},
		{name: "day_too_low", r: "y", args: args{offsetStr: "0600 245959"}, wantErr: true},
		{name: "month_exceed", r: "y", args: args{offsetStr: "1300 245959"}, wantErr: true},
		{name: "month_too_low", r: "y", args: args{offsetStr: "0000 245959"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.parseTimeoffset(tt.args.offsetStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("WhenRotate.parseTimeoffset() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WhenRotate.parseTimeoffset() = %v, want %v", got, tt.want)
			}
		})
	}
}
