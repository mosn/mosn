package router

import (
	"fmt"
	"reflect"
	"testing"

	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

func TestGenerateHashedValue(t *testing.T) {

	test := types.GenerateHashedValue("test")
	fmt.Println(test)

	type args struct {
		input string
	}
	tests := []struct {
		name string
		args args
		want types.HashedValue
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := types.GenerateHashedValue(tt.args.input); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GenerateHashedValue() = %v, want %v", got, tt.want)
			}
		})
	}
}





func TestGetEnvoyLBMetaData(t *testing.T) {
	
	header := v2.HeaderMatcher{
		Name:"service",
		Value:"com.alipay.rpc.common.service.facade.SampleService:1.0",
	}
	
	var envoyvalue = map[string]interface{} {"label": "gray","stage":"pre-release"}
	
	var value = map[string]interface{}{"envoy.lb":envoyvalue}
	
	routerV2 := v2.Router{
		Match:v2.RouterMatch{
			Headers:[]v2.HeaderMatcher{header},
		},
		
		Route:v2.RouteAction{
			ClusterName:"testclustername",
			MetadataMatch:v2.Metadata{
				"filter_metadata":value,
			},
		},
	}
	type args struct {
		route *v2.Router
	}
	
	tests := []struct {
		name string
		args args
		want map[string]interface{}
	}{
		{
			name:"testcase1",
			args:args{
				route:&routerV2,
				},
			want:map[string]interface{}{
				"label": "gray","stage":"pre-release",
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetEnvoyLBMetaData(tt.args.route); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetEnvoyLBMetaData() = %v, want %v", got, tt.want)
			}
		})
	}
}
