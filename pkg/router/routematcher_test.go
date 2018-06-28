package router

import (
	"reflect"
	"testing"

	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

func TestNewRouteMatcher(t *testing.T) {
	type args struct {
		config interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    types.Routers
		wantErr bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewRouteMatcher(tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRouteMatcher() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewRouteMatcher() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRouteMatcher_Route(t *testing.T) {
	type args struct {
		headers     map[string]string
		randomValue uint64
	}
	tests := []struct {
		name string
		rm   *RouteMatcher
		args args
		want types.Route
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rm.Route(tt.args.headers, tt.args.randomValue); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RouteMatcher.Route() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRouteMatcher_findVirtualHost(t *testing.T) {
	type args struct {
		headers map[string]string
	}
	tests := []struct {
		name string
		rm   *RouteMatcher
		args args
		want types.VirtualHost
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rm.findVirtualHost(tt.args.headers); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RouteMatcher.findVirtualHost() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRouteMatcher_findWildcardVirtualHost(t *testing.T) {
	type args struct {
		host string
	}
	tests := []struct {
		name string
		rm   *RouteMatcher
		args args
		want types.VirtualHost
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rm.findWildcardVirtualHost(tt.args.host); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RouteMatcher.findWildcardVirtualHost() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRouteMatcher_AddRouter(t *testing.T) {
	type args struct {
		routerName string
	}
	tests := []struct {
		name string
		rm   *RouteMatcher
		args args
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.rm.AddRouter(tt.args.routerName)
		})
	}
}

func TestRouteMatcher_DelRouter(t *testing.T) {
	type args struct {
		routerName string
	}
	tests := []struct {
		name string
		rm   *RouteMatcher
		args args
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.rm.DelRouter(tt.args.routerName)
		})
	}
}
