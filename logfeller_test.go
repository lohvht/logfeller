// package logfeller implements a library for writing to and rotating files
// based on a timed schedule.
package logfeller

import (
	"reflect"
	"testing"
)

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
