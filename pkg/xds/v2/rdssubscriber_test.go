package v2

import (
	"reflect"
	"testing"

	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
)

func Test_handleRoutesResp(t *testing.T) {
	type args struct {
		resp *envoy_api_v2.DiscoveryResponse
	}
	tests := []struct {
		name string
		args args
		want []*envoy_api_v2.RouteConfiguration
	}{
		{
			name: "case1",
			args: args{
				resp: &envoy_api_v2.DiscoveryResponse{},
			},
			want: []*envoy_api_v2.RouteConfiguration{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var client ClientV2
			if got := client.handleRoutesResp(tt.args.resp); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("handleRoutesResp() = %v, want %v", got, tt.want)
			}
		})
	}
}
