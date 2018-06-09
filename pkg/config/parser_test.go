package config

import (
	"reflect"
	"testing"
	"time"

	"encoding/json"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
)

func TestParseClusterHealthCheckConf(t *testing.T) {

	healthCheckConfigStr := `{
	
		  "protocol": "SofaRpc",
          "timeout": "90s",
          "healthy_threshold": 2,
          "unhealthy_threshold": 2,
          "interval": "5s",
          "interval_jitter": 0,
          "check_path": ""
    }`

	var ccc ClusterHealthCheckConfig

	json.Unmarshal([]byte(healthCheckConfigStr), &ccc)

	want := v2.HealthCheck{
		Protocol:           "SofaRpc",
		Timeout:            90 * time.Second,
		HealthyThreshold:   2,
		UnhealthyThreshold: 2,
		Interval:           5 * time.Second,
		IntervalJitter:     0,
		CheckPath:          "",
		ServiceName:        "",
	}

	type args struct {
		c *ClusterHealthCheckConfig
	}
	tests := []struct {
		name string
		args args
		want v2.HealthCheck
	}{
		// TODO: Add test cases.
		{
			name: "test1",
			args: args{
				c: &ccc,
			},
			want: want,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseClusterHealthCheckConf(tt.args.c); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseClusterHealthCheckConf() = %v, want %v", got, tt.want)
			}
		})
	}
}
