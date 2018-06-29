package config

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"
)

func Test_convertClusterHealthCheck(t *testing.T) {

	argS := v2.HealthCheck{
		Protocol:           "SofaRpc",
		Timeout:            90 * time.Second,
		Interval:           5 * time.Second,
		IntervalJitter:     0,
		HealthyThreshold:   2,
		UnhealthyThreshold: 2,
		CheckPath:          "",
		ServiceName:        "",
	}

	wantS := ClusterHealthCheckConfig{
		Protocol:           "SofaRpc",
		Timeout:            DurationConfig{90 * time.Second},
		HealthyThreshold:   2,
		UnhealthyThreshold: 2,
		Interval:           DurationConfig{5 * time.Second},
		IntervalJitter:     DurationConfig{0},
		CheckPath:          "",
		ServiceName:        "",
	}
	
	value,_ := json.Marshal(wantS)
	fmt.Print(string(value))
	
	type args struct {
		cchc v2.HealthCheck
	}

	tests := []struct {
		name string
		args args
		want ClusterHealthCheckConfig
	}{
		{
			name: "test1",
			args: args{
				argS,
			},
			want: wantS,
		},

		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := convertClusterHealthCheck(tt.args.cchc); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("convertClusterHealthCheck() = %v, want %v", got, tt.want)
			}
		})
		
	}
}
