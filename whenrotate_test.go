/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

package logfeller

import (
	"reflect"
	"testing"
	"time"
)

func TestWhenRotate_valid(t *testing.T) {
	tests := []struct {
		name    string
		r       WhenRotate
		wantErr bool
	}{
		{name: "hourly_lower", r: "h"},
		{name: "daily_lower", r: "d"},
		{name: "monthly_lower", r: "m"},
		{name: "yearly_lower", r: "y"},
		{name: "invalid_singlechar", r: "a", wantErr: true},
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
		want timeSchedule
	}{
		{name: "hourly_lower", r: "h", want: timeSchedule{}},
		{name: "daily_lower", r: "d", want: timeSchedule{}},
		{name: "monthly_lower", r: "m", want: timeSchedule{day: 1}},
		{name: "yearly_lower", r: "y", want: timeSchedule{day: 1, month: 1}},
		{name: "invalid_singlechar", r: "a", want: timeSchedule{day: 1, month: 1}},
		{name: "invalid_multiplechar", r: "hour", want: timeSchedule{day: 1, month: 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.baseRotateTime(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WhenRotate.baseRotateTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWhenRotate_parsetimeSchedule(t *testing.T) {
	type args struct {
		offsetStr string
	}
	tests := []struct {
		name    string
		r       WhenRotate
		args    args
		want    timeSchedule
		wantErr bool
	}{
		{name: "hourly", r: "h", args: args{offsetStr: "14:45"}, want: timeSchedule{minute: 14, second: 45}},
		{name: "daily", r: "d", args: args{offsetStr: "1914:45"}, want: timeSchedule{hour: 19, minute: 14, second: 45}},
		{name: "monthly", r: "m", args: args{offsetStr: "15 1914:45"}, want: timeSchedule{day: 15, hour: 19, minute: 14, second: 45}},
		{name: "yearly", r: "y", args: args{offsetStr: "0615 1914:45"}, want: timeSchedule{month: 6, day: 15, hour: 19, minute: 14, second: 45}},
		{name: "when_error", r: "hour", wantErr: true},
		{name: "hourly_format_invalid", r: "h", args: args{offsetStr: "114451"}, wantErr: true},
		{name: "daily_format_invalid", r: "D", args: args{offsetStr: "1 114451"}, wantErr: true},
		{name: "monthly_format_invalid", r: "m", args: args{offsetStr: "111 114451"}, wantErr: true},
		{name: "yearly_format_invalid", r: "Y", args: args{offsetStr: "31111 114451"}, wantErr: true},
		{name: "second_exeed", r: "y", args: args{offsetStr: "0615 1900:61"}, wantErr: true},
		{name: "minute_exeed", r: "y", args: args{offsetStr: "0615 1961:59"}, wantErr: true},
		{name: "hour_exeed", r: "y", args: args{offsetStr: "0615 2459:59"}, wantErr: true},
		{name: "day_exeed", r: "y", args: args{offsetStr: "0632 2459:59"}, wantErr: true},
		{name: "day_too_low", r: "y", args: args{offsetStr: "0600 2459:59"}, wantErr: true},
		{name: "month_exceed", r: "y", args: args{offsetStr: "1300 2459:59"}, wantErr: true},
		{name: "month_too_low", r: "y", args: args{offsetStr: "0000 2459:59"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.parseTimeSchedule(tt.args.offsetStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("WhenRotate.parsetimeSchedule() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WhenRotate.parsetimeSchedule() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWhenRotate_nearestScheduledTime(t *testing.T) {
	type args struct {
		currentTime time.Time
		sch         timeSchedule
	}
	tests := []struct {
		name string
		r    WhenRotate
		args args
		want time.Time
	}{
		{
			name: "schedule_at_30min45s_hourly_currtime_before",
			r:    "h",
			args: args{
				currentTime: time.Date(2010, 1, 2, 23, 12, 0, 0, time.Local),
				sch:         timeSchedule{minute: 30, second: 45},
			},
			want: time.Date(2010, 1, 2, 23, 30, 45, 0, time.Local),
		},
		{
			name: "schedule_at_15min24s_hourly_currtime_after",
			r:    "h",
			args: args{
				currentTime: time.Date(2010, 1, 2, 23, 59, 0, 0, time.Local),
				sch:         timeSchedule{minute: 15, second: 24},
			},
			want: time.Date(2010, 1, 2, 23, 15, 24, 0, time.Local),
		},
		{
			name: "schedule_at_0min_hourly_currtime",
			r:    "h",
			args: args{
				currentTime: time.Date(2010, 1, 2, 23, 59, 0, 0, time.Local),
				sch:         timeSchedule{},
			},
			want: time.Date(2010, 1, 2, 23, 0, 0, 0, time.Local),
		},
		{
			name: "schedule_at_1230:20_daily_currtime_before",
			r:    "d",
			args: args{
				currentTime: time.Date(2010, 1, 2, 5, 59, 0, 0, time.Local),
				sch:         timeSchedule{hour: 12, minute: 30, second: 20},
			},
			want: time.Date(2010, 1, 2, 12, 30, 20, 0, time.Local),
		},
		{
			name: "schedule_at_0930:22_daily_currtime_after",
			r:    "d",
			args: args{
				currentTime: time.Date(2010, 1, 2, 23, 59, 0, 0, time.Local),
				sch:         timeSchedule{hour: 9, minute: 30, second: 22},
			},
			want: time.Date(2010, 1, 2, 9, 30, 22, 0, time.Local),
		},
		{
			name: "schedule_at_0000_daily_currtime",
			r:    "d",
			args: args{
				currentTime: time.Date(2010, 1, 2, 23, 59, 0, 0, time.Local),
				sch:         timeSchedule{},
			},
			want: time.Date(2010, 1, 2, 0, 0, 0, 0, time.Local),
		},
		{
			name: "schedule_at_15thday_1230:20_monthly_currtime_before",
			r:    "m",
			args: args{
				currentTime: time.Date(2010, 1, 2, 5, 59, 0, 0, time.Local),
				sch:         timeSchedule{day: 15, hour: 12, minute: 30, second: 20},
			},
			want: time.Date(2010, 1, 15, 12, 30, 20, 0, time.Local),
		},
		{
			name: "schedule_at_7thday_0930:22_monthly_currtime_after",
			r:    "m",
			args: args{
				currentTime: time.Date(2010, 1, 20, 23, 59, 0, 0, time.Local),
				sch:         timeSchedule{day: 7, hour: 9, minute: 30, second: 22},
			},
			want: time.Date(2010, 1, 7, 9, 30, 22, 0, time.Local),
		},
		{
			name: "schedule_at_1st_monthly_currtime",
			r:    "m",
			args: args{
				currentTime: time.Date(2010, 1, 20, 23, 59, 0, 0, time.Local),
				sch:         timeSchedule{day: 1},
			},
			want: time.Date(2010, 1, 1, 0, 0, 0, 0, time.Local),
		},
		{
			name: "schedule_at_october_15thday_1230:20_yearly_currtime_before",
			r:    "y",
			args: args{
				currentTime: time.Date(2010, 1, 2, 5, 59, 0, 0, time.Local),
				sch:         timeSchedule{month: 10, day: 15, hour: 12, minute: 30, second: 20},
			},
			want: time.Date(2010, 10, 15, 12, 30, 20, 0, time.Local),
		},
		{
			name: "schedule_at_january_7thday_0930:22_yearly_currtime_after",
			r:    "y",
			args: args{
				currentTime: time.Date(2010, 8, 20, 23, 59, 0, 0, time.Local),
				sch:         timeSchedule{month: 1, day: 7, hour: 9, minute: 30, second: 22},
			},
			want: time.Date(2010, 1, 7, 9, 30, 22, 0, time.Local),
		},
		{
			name: "schedule_at_january_1st_yearly_currtime",
			r:    "y",
			args: args{
				currentTime: time.Date(2010, 1, 20, 23, 59, 0, 0, time.Local),
				sch:         timeSchedule{month: 1, day: 1},
			},
			want: time.Date(2010, 1, 1, 0, 0, 0, 0, time.Local),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.nearestScheduledTime(tt.args.currentTime, tt.args.sch); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WhenRotate.nearestScheduledTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWhenRotate_AddTime(t *testing.T) {
	type args struct {
		t time.Time
		n int
	}
	tests := []struct {
		name string
		r    WhenRotate
		args args
		want time.Time
	}{
		{
			name: "add_1_hour",
			r:    "h",
			args: args{
				t: time.Date(2010, 8, 20, 20, 59, 0, 0, time.Local),
				n: 1,
			},
			want: time.Date(2010, 8, 20, 21, 59, 0, 0, time.Local),
		},
		{
			name: "add_1_hour_to_next_day",
			r:    "h",
			args: args{
				t: time.Date(2010, 8, 20, 23, 59, 0, 0, time.Local),
				n: 1,
			},
			want: time.Date(2010, 8, 21, 0, 59, 0, 0, time.Local),
		},
		{
			name: "minus_1_hour",
			r:    "h",
			args: args{
				t: time.Date(2010, 8, 20, 20, 59, 0, 0, time.Local),
				n: -1,
			},
			want: time.Date(2010, 8, 20, 19, 59, 0, 0, time.Local),
		},
		{
			name: "minus_1_hour_to_prev_day",
			r:    "h",
			args: args{
				t: time.Date(2010, 8, 20, 0, 59, 0, 0, time.Local),
				n: -1,
			},
			want: time.Date(2010, 8, 19, 23, 59, 0, 0, time.Local),
		},
		{
			name: "add_1_day",
			r:    "d",
			args: args{
				t: time.Date(2010, 8, 20, 20, 59, 0, 0, time.Local),
				n: 1,
			},
			want: time.Date(2010, 8, 21, 20, 59, 0, 0, time.Local),
		},
		{
			name: "add_1_day_to_next_month",
			r:    "d",
			args: args{
				t: time.Date(2010, 8, 31, 23, 59, 0, 0, time.Local),
				n: 1,
			},
			want: time.Date(2010, 9, 1, 23, 59, 0, 0, time.Local),
		},
		{
			name: "minus_1_day",
			r:    "d",
			args: args{
				t: time.Date(2010, 8, 20, 20, 59, 0, 0, time.Local),
				n: -1,
			},
			want: time.Date(2010, 8, 19, 20, 59, 0, 0, time.Local),
		},
		{
			name: "minus_1_day_to_prev_month",
			r:    "d",
			args: args{
				t: time.Date(2010, 8, 1, 0, 59, 0, 0, time.Local),
				n: -1,
			},
			want: time.Date(2010, 7, 31, 0, 59, 0, 0, time.Local),
		},
		{
			name: "add_1_month",
			r:    "m",
			args: args{
				t: time.Date(2010, 8, 20, 20, 59, 0, 0, time.Local),
				n: 1,
			},
			want: time.Date(2010, 9, 20, 20, 59, 0, 0, time.Local),
		},
		{
			name: "add_1_month_to_next_year",
			r:    "m",
			args: args{
				t: time.Date(2010, 12, 31, 23, 59, 0, 0, time.Local),
				n: 1,
			},
			want: time.Date(2011, 1, 31, 23, 59, 0, 0, time.Local),
		},
		{
			name: "minus_1_month",
			r:    "m",
			args: args{
				t: time.Date(2010, 8, 20, 20, 59, 0, 0, time.Local),
				n: -1,
			},
			want: time.Date(2010, 7, 20, 20, 59, 0, 0, time.Local),
		},
		{
			name: "minus_1_month_to_prev_year",
			r:    "m",
			args: args{
				t: time.Date(2010, 1, 1, 0, 59, 0, 0, time.Local),
				n: -1,
			},
			want: time.Date(2009, 12, 1, 0, 59, 0, 0, time.Local),
		},
		{
			name: "add_1_year",
			r:    "y",
			args: args{
				t: time.Date(2010, 8, 20, 20, 59, 0, 0, time.Local),
				n: 1,
			},
			want: time.Date(2011, 8, 20, 20, 59, 0, 0, time.Local),
		},
		{
			name: "minus_1_year",
			r:    "y",
			args: args{
				t: time.Date(2010, 8, 20, 20, 59, 0, 0, time.Local),
				n: -1,
			},
			want: time.Date(2009, 8, 20, 20, 59, 0, 0, time.Local),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.addTime(tt.args.t, tt.args.n); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WhenRotate.AddTime() = %v, want %v", got, tt.want)
			}
		})
	}
}
