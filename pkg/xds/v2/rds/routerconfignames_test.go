package rds

import (
	"reflect"
	"testing"
)

func Test_AppendRouterName(t *testing.T) {
	type args struct {
		routerName  string
	}
	tests := []struct {
		name string
		args args
		want map[string]bool
	}{
		{
			name: "case1",
			args: args{
				routerName:   "http.80",
			},
			want: map[string]bool{"http.80":true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AppendRouterName(tt.args.routerName)
			if got := routerNames; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AppendRouterName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_GetRouterNames(t *testing.T) {
	routerNames = make(map[string]bool)
	routerNames["http.80"] = true

	tests := []struct {
		name string
		want []string
	}{
		{
			name: "case1",
			want: []string{"http.80"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetRouterNames(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetRouterNames() = %v, want %v", got, tt.want)
			}
		})
	}
}
