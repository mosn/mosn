package router

import (
	"reflect"
	"testing"

	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

func TestNewVirtualHostImpl(t *testing.T) {
	type args struct {
		virtualHost      *v2.VirtualHost
		validateClusters bool
	}
	tests := []struct {
		name string
		args args
		want *VirtualHostImpl
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewVirtualHostImpl(tt.args.virtualHost, tt.args.validateClusters); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewVirtualHostImpl() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVirtualHostImpl_Name(t *testing.T) {
	tests := []struct {
		name string
		vh   *VirtualHostImpl
		want string
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.vh.Name(); got != tt.want {
				t.Errorf("VirtualHostImpl.Name() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVirtualHostImpl_CorsPolicy(t *testing.T) {
	tests := []struct {
		name string
		vh   *VirtualHostImpl
		want types.CorsPolicy
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.vh.CorsPolicy(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("VirtualHostImpl.CorsPolicy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVirtualHostImpl_RateLimitPolicy(t *testing.T) {
	tests := []struct {
		name string
		vh   *VirtualHostImpl
		want types.RateLimitPolicy
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.vh.RateLimitPolicy(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("VirtualHostImpl.RateLimitPolicy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVirtualHostImpl_GetRouteFromEntries(t *testing.T) {
	type args struct {
		headers     map[string]string
		randomValue uint64
	}
	tests := []struct {
		name string
		vh   *VirtualHostImpl
		args args
		want types.Route
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.vh.GetRouteFromEntries(tt.args.headers, tt.args.randomValue); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("VirtualHostImpl.GetRouteFromEntries() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVirtualClusterEntry_VirtualClusterName(t *testing.T) {
	tests := []struct {
		name string
		vce  *VirtualClusterEntry
		want string
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.vce.VirtualClusterName(); got != tt.want {
				t.Errorf("VirtualClusterEntry.VirtualClusterName() = %v, want %v", got, tt.want)
			}
		})
	}
}
