/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package config

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/alipay/sofa-mosn/internal/api/v2"
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
			if got := parseClusterHealthCheckConf(tt.args.c); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseClusterHealthCheckConf() = %v, want %v", got, tt.want)
			}
		})
	}
}
