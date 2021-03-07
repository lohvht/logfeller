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
		{name: "hourly", r: "h", args: args{offsetStr: "1445"}, want: timeSchedule{minute: 14, second: 45}},
		{name: "daily", r: "d", args: args{offsetStr: "191445"}, want: timeSchedule{hour: 19, minute: 14, second: 45}},
		{name: "monthly", r: "m", args: args{offsetStr: "15 191445"}, want: timeSchedule{day: 15, hour: 19, minute: 14, second: 45}},
		{name: "yearly", r: "y", args: args{offsetStr: "0615 191445"}, want: timeSchedule{month: 6, day: 15, hour: 19, minute: 14, second: 45}},
		{name: "when_error", r: "hour", wantErr: true},
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
