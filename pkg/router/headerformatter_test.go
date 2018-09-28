package router

import (
	"reflect"
	"testing"

)

func Test_getHeaderFormatter(t *testing.T) {
	type args struct {
		value  string
		append bool
	}
	tests := []struct {
		name string
		args args
		want headerFormatter
	}{
		{
			name: "case1",
			args: args{
				value:  "demo",
				append: false,
			},
			want: &plainHeaderFormatter{
				isAppend:    false,
				staticValue: "demo",
			},
		},
		{
			name: "case2",
			args: args{
				value:  "%address%",
				append: false,
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getHeaderFormatter(tt.args.value, tt.args.append); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getHeaderFormatter(value string, append bool) = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_plainHeaderFormatter_append(t *testing.T) {
	formatter := plainHeaderFormatter{
		isAppend:    false,
		staticValue: "demo",
	}
	tests := []struct {
		name string
		want bool
	}{
		{
			name: "case1",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatter.append(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("(f *plainHeaderFormatter) append() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_plainHeaderFormatter_format(t *testing.T) {
	formatter := plainHeaderFormatter{
		isAppend:    false,
		staticValue: "demo",
	}

	tests := []struct {
		name string
		want string
	}{
		{
			name: "case1",
			want: "demo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatter.format(nil); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("(f *plainHeaderFormatter) format(requestInfo types.RequestInfo) = %v, want %v", got, tt.want)
			}
		})
	}
}
