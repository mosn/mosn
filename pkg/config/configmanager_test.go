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
	"reflect"
	"testing"
	"time"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
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
